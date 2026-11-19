import { lazy, Suspense, useState, useEffect, useRef } from "react";
import { Link, Navigate, useParams } from "react-router-dom";
import { Weekly } from "./history";
import { directions } from "./survey";
import { authorized } from "./api";
import "./support.css";

const Housing = lazy(() => import("./housing"));
const descriptions: Record<string, string> = {
  housing: "Посуточная аренда в Алматы",
  jobs: "Вакансии по вашим навыкам",
  psychologist: "Подготовить обращение",
  social: "Волонтёрские организации",
  emergency: "Номера служб Казахстана",
};
export function Home() {
  return (
    <section className="week">
      <nav className="support-nav">
        <Link to="/profile">Профиль</Link>
        <Link to="/history">История по неделям</Link>
      </nav>
      <h1>Какая помощь вам нужна?</h1>
      <Weekly />
      <div className="support-grid">
        {Object.entries(descriptions).map(([k, desc]) => (
          <Link key={k} to={"/support/" + k}>
            <strong>{directions[k]}</strong>
            <small>{desc}</small>
          </Link>
        ))}
      </div>
      <p>
        <Link to="/survey">Пройти опрос «Моя неделя» →</Link>
      </p>
    </section>
  );
}

function Jobs() {
  const [text, setText] = useState(""),
    [confirmed, setConfirmed] = useState(""),
    [busy, setBusy] = useState(false),
    [failed, setFailed] = useState(false);
  const [items, setItems] = useState<
    { title: string; employer: string; url: string }[] | null
  >(null);
  async function search() {
    const query = text.trim();
    if (!query || busy) return;
    setConfirmed(query);
    setBusy(true);
    setFailed(false);
    setItems(null);
    try {
      const result = await authorized<{ items: NonNullable<typeof items> }>(
        "/support/vacancies",
        { text: query, area: "160" },
      );
      setItems(result.items ?? []);
    } catch {
      setFailed(true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void search();
        }}
      >
        <label>
          Навыки или профессия
          <input
            placeholder="Например, ремонт техники"
            value={text}
            maxLength={160}
            disabled={busy}
            onChange={(e) => setText(e.target.value)}
          />
        </label>
        <button className="primary" disabled={!text.trim() || busy}>
          {busy ? "Ищем вакансии…" : "Искать на HeadHunter"}
        </button>
      </form>
      {failed && (
        <p role="alert">
          HeadHunter сейчас недоступен. Повторите поиск или откройте hh.kz.
        </p>
      )}
      {items?.length === 0 && (
        <p role="status">По этим словам в Алматы вакансий не найдено.</p>
      )}
      {items?.map((v) => (
        <article className="history-entry" key={v.url}>
          <a href={v.url} target="_blank" rel="noreferrer">
            {v.title}
          </a>
          <p>{v.employer}</p>
        </article>
      ))}
      {confirmed && (
        <p>
          <a
            href={
              "https://hh.kz/search/vacancy?" +
              new URLSearchParams({ area: "160", text: confirmed })
            }
            target="_blank"
            rel="noreferrer"
          >
            Открыть этот поиск на hh.kz ↗
          </a>
        </p>
      )}
    </>
  );
}
function Draft({ social }: { social: boolean }) {
  const [draft, setDraft] = useState(""),
    [notice, setNotice] = useState(""),
    edited = useRef(false);
  useEffect(() => {
    const controller = new AbortController(),
      signal = controller.signal;
    async function fill() {
      const { id } = await authorized<{ id: string }>(
        "/surveys/latest",
        undefined,
        signal,
      );
      if (!id) return;
      const result = await authorized<{ status: string; summary: string }>(
        "/assessments/" + encodeURIComponent(id),
        undefined,
        signal,
      );
      if (!signal.aborted && !edited.current && result.status === "completed")
        setDraft(result.summary ?? "");
    }
    void fill().catch(() => {});
    return () => controller.abort();
  }, []);
  return (
    <>
      <label>
        Что вы хотите рассказать?
        <textarea
          value={draft}
          onChange={(e) => {
            edited.current = true;
            setDraft(e.target.value);
            setNotice("");
          }}
        />
      </label>
      <button
        className="text"
        disabled={!draft.trim()}
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(draft.trim());
            setNotice("Обращение скопировано.");
          } catch {
            setNotice("Выделите текст и скопируйте его вручную.");
          }
        }}
      >
        Скопировать обращение
      </button>
      {notice && <p role="status">{notice}</p>}
      {social ? (
        <ul className="contacts">
          <li>
            <a href="https://t.me/BIRGEMIZ2020">Написать Birgemiz в Telegram</a>
          </li>
          <li>
            <a href="https://qazvolunteer.kz/ru/contacts">
              Региональные контакты Qazvolunteer
            </a>
          </li>
          <li>
            <a href="https://dobrye-dela.kz/">Добрые дела Алматы</a>
          </li>
          <li>
            <a href="https://qazvolunteer.kz/ru">
              Найти волонтёрскую организацию
            </a>
          </li>
        </ul>
      ) : (
        <ul className="contacts">
          <li>
            <a href="tel:1303">1303 · Центр психического здоровья Алматы</a>
          </li>
          <li>
            <a href="tel:150">150 · Помощь детям, молодёжи и семьям</a>
          </li>
          <li>
            <a href="https://www.gov.kz/org/almaty-densaulyk/press/news/details/1220242?lang=ru">
              Информация городской службы
            </a>
          </li>
          <li>
            <Link to="/survey">Перейти к опросу</Link>
          </li>
        </ul>
      )}
    </>
  );
}
export function Support({ signedIn }: { signedIn: boolean }) {
  const { kind = "" } = useParams();
  if (!Object.hasOwn(descriptions, kind))
    return <Navigate to="/home" replace />;
  if (!signedIn && kind !== "emergency")
    return <Navigate to="/login" replace />;
  return (
    <section className="week">
      <Link to={signedIn ? "/home" : "/login"}>
        {signedIn ? "← На главную" : "← Войти"}
      </Link>
      <h1>{directions[kind]}</h1>
      {kind === "housing" ? (
        <Suspense fallback={<p>Загружаем районы…</p>}>
          <Housing />
        </Suspense>
      ) : kind === "jobs" ? (
        <Jobs />
      ) : kind === "emergency" ? (
        <ul className="contacts">
          {Object.entries({
            112: "Единая экстренная служба",
            101: "Пожарные и спасатели",
            102: "Полиция",
            103: "Скорая медицинская помощь",
            104: "Аварийная газовая служба",
          }).map(([n, label]) => (
            <li key={n}>
              <a href={"tel:" + n}>
                {n} · {label}
              </a>
            </li>
          ))}
        </ul>
      ) : (
        <Draft key={kind} social={kind === "social"} />
      )}
    </section>
  );
}
