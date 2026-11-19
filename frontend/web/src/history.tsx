import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { authorized } from "./api";
import { directions, topics } from "./survey";

interface Entry {
  id: string;
  completed_at: number;
  version: number;
  percentage: number | null;
  ratings: Record<string, number> | null;
  directions: string[] | null;
}
interface Page {
  entries: Entry[];
  next_cursor: string;
  next_due: number;
  reminder_due: boolean;
}
const date = (seconds: number) =>
  new Date(seconds * 1000).toLocaleString("ru-RU", {
    dateStyle: "long",
    timeStyle: "short",
  });
const signed = (n: number) => (n > 0 ? "+" + n : String(n));
export function Weekly({ history = false }: { history?: boolean }) {
  const [pages, setPages] = useState<Page[]>([]);
  const [cursor, setCursor] = useState("");
  const [busy, setBusy] = useState(true),
    [failed, setFailed] = useState(false),
    [attempt, retry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setBusy(true);
    setFailed(false);
    authorized<Page>(
      "/weekly/history" +
        (cursor ? "?before=" + encodeURIComponent(cursor) : ""),
      undefined,
      controller.signal,
    )
      .then((p) => {
        if (!controller.signal.aborted)
          setPages((previous) => (cursor ? [...previous, p] : [p]));
      })
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [cursor, attempt]);
  const latest = pages[0],
    next = pages.at(-1)?.next_cursor;
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!latest?.next_due) return;
    const delay = latest.next_due * 1000 - Date.now();
    if (delay <= 0) {
      setNow(Date.now());
      return;
    }
    const timer = setTimeout(() => setNow(Date.now()), delay + 100);
    return () => clearTimeout(timer);
  }, [latest]);
  const due =
    latest &&
    (latest.reminder_due ||
      (latest.next_due > 0 && latest.next_due * 1000 <= now));
  const entries = pages.flatMap((p) => p.entries);
  return (
    <>
      <div className="weekly-invite">
        {due ? (
          <Link to="/survey">Пора пройти опрос о неделе →</Link>
        ) : history && latest?.next_due ? (
          <span>Следующий опрос · {date(latest.next_due)}</span>
        ) : null}
      </div>
      {history && (
        <>
          {entries.map((e, i) => {
            const older = entries[i + 1],
              prior =
                e.version === 2 && older?.version === 2 ? older : undefined;
            const selected = e.directions ?? [],
              old = prior?.directions ?? [];
            const names = (ids: string[]) =>
              ids.map((k) => directions[k] ?? k).join(", ");
            return (
              <article className="history-entry" key={e.id}>
                <h2>{date(e.completed_at)}</h2>
                {e.percentage != null && (
                  <>
                    <strong>Самооценка · {e.percentage}%</strong>
                    <progress
                      value={e.percentage}
                      max={100}
                      aria-label="Самооценка"
                    />
                    {prior?.percentage != null && (
                      <p>
                        Изменение · {signed(e.percentage - prior.percentage)} п.
                        п.
                      </p>
                    )}
                  </>
                )}
                {Object.entries(e.ratings ?? {}).map(([k, n]) => (
                  <p key={k}>
                    {topics[k] ?? k} · {n}/4{" "}
                    {prior?.ratings?.[k] != null &&
                      "(" + signed(n - prior.ratings[k]) + ")"}
                  </p>
                ))}
                {selected.length > 0 && <p>Рекомендации · {names(selected)}</p>}
                {prior && selected.some((k) => !old.includes(k)) && (
                  <p>
                    Добавились ·{" "}
                    {names(selected.filter((k) => !old.includes(k)))}
                  </p>
                )}
                {prior && old.some((k) => !selected.includes(k)) && (
                  <p>
                    Больше не предложены ·{" "}
                    {names(old.filter((k) => !selected.includes(k)))}
                  </p>
                )}
                <Link to={"/assessment/" + encodeURIComponent(e.id)}>
                  Открыть результат
                </Link>
              </article>
            );
          })}
          {busy && <p role="status">Загружаем историю…</p>}
          {!busy && !failed && !entries.length && (
            <p>Здесь появятся результаты завершённых опросов.</p>
          )}
          {!busy && !failed && next && (
            <button className="text" onClick={() => setCursor(next)}>
              Более ранние опросы
            </button>
          )}
        </>
      )}
      {failed && (
        <button className="text" onClick={() => retry(attempt + 1)}>
          Повторить загрузку истории
        </button>
      )}
    </>
  );
}
export function History() {
  return (
    <section className="week">
      <Link to="/home">← На главную</Link>
      <h1>Динамика по неделям</h1>
      <Weekly history />
    </section>
  );
}
