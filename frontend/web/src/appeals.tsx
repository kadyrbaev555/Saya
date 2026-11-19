import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { authorized, publicGet, post } from "./api";

type Scope = "public" | "mine" | "review";
interface Appeal {
  id: string;
  description: string;
  state?: string;
  reported?: boolean;
  card_ready: boolean;
}
interface List {
  items: Appeal[];
  checkout_available: boolean;
}
const labels: Record<string, string> = {
  pending: "На модерации",
  published: "Опубликована",
  declined: "Отклонена",
  withdrawn: "Снята",
  expired: "Скрыта после нового опроса",
};
export function Appeals({
  scope,
  signedIn,
}: {
  scope: Scope;
  signedIn: boolean;
}) {
  const [list, setList] = useState<List>(),
    [failed, setFailed] = useState(false),
    [busy, setBusy] = useState(false),
    [attempt, reload] = useState(0),
    [notice, setNotice] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    setFailed(false);
    setList(undefined);
    const load =
      scope === "public"
        ? publicGet<List>("/appeals", controller.signal)
        : authorized<List>("/appeals/" + scope, undefined, controller.signal);
    load
      .then((v) => {
        if (!controller.signal.aborted) setList(v);
      })
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, [scope, attempt]);
  const active = useRef<AbortController | null>(null);
  useEffect(() => () => active.current?.abort(), []);
  async function act(id: string, action: string, amount?: number) {
    if (active.current) return;
    const controller = new AbortController();
    active.current = controller;
    setBusy(true);
    setNotice("");
    try {
      const path = "/appeals/" + encodeURIComponent(id) + "/" + action;
      if (action === "checkout" || action === "payout-card") {
        const result =
          action === "checkout"
            ? await post<{ url: string }>(path, { amount })
            : await authorized<{ url: string }>(path, {});
        if (controller.signal.aborted) return;
        const url = new URL(result.url);
        if (url.protocol !== "https:" || url.username || url.password)
          throw new Error();
        window.location.assign(url.href);
      } else {
        if (scope === "public") await post<void>(path, {});
        else await authorized<void>(path, {});
        if (controller.signal.aborted) return;
        setNotice(
          action === "report"
            ? "Жалоба передана на проверку."
            : action === "withdraw"
              ? "Анкета снята."
              : "Решение сохранено.",
        );
        reload((n) => n + 1);
      }
    } catch {
      if (!controller.signal.aborted)
        setNotice("Действие не выполнено. Обновите список и попробуйте снова.");
    } finally {
      active.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }
  return (
    <section className="week">
      <Link
        to={signedIn ? (scope === "public" ? "/home" : "/profile") : "/login"}
      >
        {signedIn
          ? scope === "public"
            ? "← На главную"
            : "← В профиль"
          : "← Войти"}
      </Link>
      <h1>
        {scope === "public"
          ? "Помочь человеку"
          : scope === "mine"
            ? "Моя анкета"
            : "Модерация анкет"}
      </h1>
      <button
        className="text"
        disabled={busy}
        onClick={() => reload((n) => n + 1)}
      >
        Обновить список
      </button>
      {notice && <p role="status">{notice}</p>}
      {failed ? (
        <p role="alert">Не удалось загрузить анкеты.</p>
      ) : !list ? (
        <p role="status">Загружаем анкеты…</p>
      ) : list.items.length === 0 ? (
        <p>
          {scope === "public"
            ? "Пока нет опубликованных анкет для помощи."
            : scope === "mine"
              ? "Вы пока не подавали анкету."
              : "Очередь проверки пуста."}
        </p>
      ) : (
        list.items.map((a) => (
          <article className="history-entry" key={a.id}>
            {scope !== "public" && <strong>{labels[a.state ?? ""]}</strong>}
            <p style={{ whiteSpace: "pre-wrap" }}>{a.description}</p>
            {scope === "mine" && (
              <>
                {list.checkout_available &&
                  ["published", "pending"].includes(a.state ?? "") && (
                    <button
                      className="text"
                      disabled={busy}
                      onClick={() => act(a.id, "payout-card")}
                    >
                      {a.card_ready
                        ? "Изменить карту у Freedom Pay"
                        : "Привязать тестовую карту у Freedom Pay"}
                    </button>
                  )}
                {a.state !== "withdrawn" && (
                  <button
                    className="text"
                    disabled={busy}
                    onClick={() => act(a.id, "withdraw")}
                  >
                    Снять анкету
                  </button>
                )}
              </>
            )}
            {scope === "review" && (
              <>
                {a.reported && <p>Поступила жалоба</p>}
                <button
                  className="text"
                  disabled={busy}
                  onClick={() => act(a.id, "approve")}
                >
                  Одобрить
                </button>
                <button
                  className="text"
                  disabled={busy}
                  onClick={() => act(a.id, "decline")}
                >
                  Отклонить
                </button>
              </>
            )}
            {scope === "public" && (
              <>
                {list.checkout_available && a.card_ready ? (
                  <fieldset>
                    <legend>Тестовый платёж через Freedom Pay</legend>
                    <div className="choices">
                      {[500, 1000, 2000].map((v) => (
                        <button
                          key={v}
                          disabled={busy}
                          onClick={() => act(a.id, "checkout", v)}
                        >
                          {v} ₸
                        </button>
                      ))}
                    </div>
                  </fieldset>
                ) : (
                  <p>Тестовый платёж для этой анкеты пока недоступен.</p>
                )}
                <button
                  className="text"
                  disabled={busy}
                  onClick={() => act(a.id, "report")}
                >
                  Пожаловаться
                </button>
              </>
            )}
          </article>
        ))
      )}
    </section>
  );
}

export function AppealEditor({
  assessment,
  summary,
}: {
  assessment: string;
  summary: string;
}) {
  const navigate = useNavigate();
  const [eligible, setEligible] = useState(false),
    [checkFailed, setCheckFailed] = useState(false),
    [attempt, retry] = useState(0);
  const [open, setOpen] = useState(false),
    [text, setText] = useState(summary),
    [consent, setConsent] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    setCheckFailed(false);
    authorized<{ entries: { id: string }[] }>(
      "/weekly/history",
      undefined,
      controller.signal,
    )
      .then((v) => setEligible(v.entries[0]?.id === assessment))
      .catch(() => {
        if (!controller.signal.aborted) setCheckFailed(true);
      });
    return () => controller.abort();
  }, [assessment, attempt]);
  const live = useRef(true);
  useEffect(() => {
    live.current = true;
    return () => {
      live.current = false;
    };
  }, []);
  async function submit() {
    if (!consent || !text.trim() || busy) return;
    setBusy(true);
    setError(false);
    try {
      await authorized(
        "/appeals/" + encodeURIComponent(assessment) + "/submit",
        { consent: true, description: text.trim() },
      );
      if (live.current) navigate("/appeals/mine");
    } catch {
      if (live.current) {
        setError(true);
        setBusy(false);
      }
    }
  }
  if (checkFailed)
    return (
      <button className="text" onClick={() => retry(attempt + 1)}>
        Проверить возможность подачи анкеты
      </button>
    );
  if (!eligible) return <>Денежная поддержка</>;
  if (!open)
    return (
      <button className="text" onClick={() => setOpen(true)}>
        Денежная поддержка · подготовить анкету
      </button>
    );
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
    >
      <h3>Анкета для помощи</h3>
      <label>
        Публичное описание
        <textarea
          autoFocus
          disabled={busy}
          value={text}
          maxLength={1200}
          onChange={(e) => {
            setText(e.target.value);
            setConsent(false);
          }}
        />
      </label>
      <small>Уберите контакты и реквизиты карты. До 1200 символов.</small>
      <label className="check">
        <input
          type="checkbox"
          checked={consent}
          disabled={busy}
          onChange={(e) => setConsent(e.target.checked)}
        />
        Согласен опубликовать этот текст для всех посетителей после модерации.
      </label>
      {error && (
        <p role="alert">
          Не удалось подать анкету. Проверьте последний результат и попробуйте
          снова.
        </p>
      )}
      <button
        className="primary"
        disabled={!consent || !text.trim() || [...text].length > 1200 || busy}
      >
        Отправить на проверку
      </button>
      <button
        className="text"
        type="button"
        disabled={busy}
        onClick={() => {
          setOpen(false);
          setConsent(false);
          setText(summary);
          setError(false);
        }}
      >
        Отмена
      </button>
    </form>
  );
}
