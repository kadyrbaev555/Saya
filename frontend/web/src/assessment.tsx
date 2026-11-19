import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Centrifuge } from "centrifuge";
import { ApiError, authorized } from "./api";
import { topics, directions } from "./survey";

interface Result {
  status: "pending" | "completed" | "failed";
  version: number;
  summary: string;
  ratings: Record<string, number> | null;
  percentage: number | null;
  directions: string[] | null;
  recommendations: string[] | null;
  disclaimer: string;
  provider: string;
}
export function LatestAssessment() {
  const [id, setId] = useState("");
  const [failed, setFailed] = useState(false);
  const [attempt, retry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setFailed(false);
    authorized<{ id: string }>("/surveys/latest", undefined, controller.signal)
      .then((result) => setId(result.id))
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, [attempt]);
  if (failed)
    return (
      <button className="text" onClick={() => retry(attempt + 1)}>
        Загрузить последний результат
      </button>
    );
  return id ? (
    <p>
      <Link to={"/assessment/" + encodeURIComponent(id)}>
        Последний результат
      </Link>
    </p>
  ) : null;
}
export function Assessment({ owner }: { owner: string }) {
  const { id = "" } = useParams();
  return <ResultPage key={owner + id} id={id} owner={owner} />;
}
function ResultPage({ id, owner }: { id: string; owner: string }) {
  const [result, setResult] = useState<Result>();
  const [error, setError] = useState(false);
  const [attempt, retry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    const signal = controller.signal;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let running = false,
      queued = false,
      done = false;
    const channel = "user:" + owner + ":assessments";
    async function token(path: string, body: object) {
      return (
        await authorized<{ token: string }>("/realtime/" + path, body, signal)
      ).token;
    }
    const ws = new URL("/connection/websocket", location.origin);
    ws.protocol = location.protocol === "https:" ? "wss:" : "ws:";
    if (import.meta.env.DEV) ws.port = "8000";
    const client = new Centrifuge(import.meta.env.VITE_WS_URL || ws.href, {
      getToken: () => token("connection-token", {}),
    });
    const subscription = client.newSubscription(channel, {
      getToken: () => token("subscription-token", { channel }),
    });
    async function load() {
      if (done || signal.aborted) return;
      if (running) {
        queued = true;
        return;
      }
      running = true;
      clearTimeout(timer);
      try {
        const current = await authorized<Result>(
          "/assessments/" + encodeURIComponent(id),
          undefined,
          signal,
        );
        if (signal.aborted) return;
        setResult(current);
        setError(false);
        done = current.status !== "pending";
        if (done) client.disconnect();
      } catch (err) {
        if (
          err instanceof ApiError &&
          [400, 401, 403, 404].includes(err.status)
        ) {
          done = true;
          client.disconnect();
        }
        if (!signal.aborted) setError(true);
      } finally {
        running = false;
        if (!signal.aborted && !done) {
          timer = setTimeout(load, queued ? 0 : 3000);
          queued = false;
        }
      }
    }
    subscription.on("subscribed", () => void load());
    subscription.on("publication", ({ data }) => {
      if (
        data?.version === 1 &&
        data.assessment_id === id &&
        ["assessment.completed", "assessment.failed"].includes(data.event)
      )
        void load();
    });
    subscription.subscribe();
    client.connect();
    void load();
    return () => {
      controller.abort();
      clearTimeout(timer);
      client.disconnect();
    };
  }, [owner, id, attempt]);
  return (
    <section className="week">
      <Link to="/profile">← В профиль</Link>
      <h1>
        {result?.status === "completed"
          ? "Результат готов"
          : result?.status === "failed"
            ? "Анализ недоступен"
            : "Результат недели"}
      </h1>
      {error && (
        <div role="alert">
          Не удалось получить результат.{" "}
          <button className="text" onClick={() => retry(attempt + 1)}>
            Повторить
          </button>
        </div>
      )}
      {(!result || result.status === "pending") && !error && (
        <p role="status">Подготовка результата…</p>
      )}
      {result && (
        <>
          <p>{result.disclaimer}</p>
          {result.status === "completed" && result.version === 2 && (
            <>
              <h2>
                {result.percentage != null
                  ? "Моя оценка недели — " + result.percentage + "%"
                  : "Мои оценки"}
              </h2>
              <p>
                {result.percentage == null
                  ? "Общий процент появится при заполнении всех пяти оценок."
                  : "Сумма пяти оценок из 20 возможных в процентах."}
              </p>
              {Object.entries(result.ratings ?? {}).map(([key, value]) => (
                <p key={key}>
                  {topics[key]} · {value}/4
                </p>
              ))}
              {result.summary && <blockquote>{result.summary}</blockquote>}
              {!!result.directions?.length && (
                <>
                  <h2>Направления поддержки</h2>
                  <ul>
                    {result.directions.map((key) => (
                      <li key={key}>
                        {key !== "financial" &&
                        Object.hasOwn(directions, key) ? (
                          <Link to={"/support/" + key}>{directions[key]}</Link>
                        ) : (
                          (directions[key] ?? key)
                        )}
                      </li>
                    ))}
                  </ul>
                </>
              )}
            </>
          )}
          {!!result.recommendations?.length && (
            <>
              <h2>Рекомендации</h2>
              <ul>
                {result.recommendations.map((text, i) => (
                  <li key={i}>{text}</li>
                ))}
              </ul>
            </>
          )}
          {result.provider === "demonstration-mock" && (
            <small>Использована демонстрационная модель.</small>
          )}
        </>
      )}
    </section>
  );
}
