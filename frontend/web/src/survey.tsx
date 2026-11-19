import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { authorized } from "./api";
import "./week.css";

export const topics: Record<string, string> = {
  housing: "Жильё",
  expenses: "Базовые расходы",
  work: "Работа",
  connection: "Общение",
  daily: "Повседневные дела",
  priority: "Приоритет",
};
export const directions: Record<string, string> = {
  housing: "Жильё",
  jobs: "Поиск работы",
  psychologist: "Психолог",
  social: "Социальная поддержка",
  financial: "Денежная поддержка",
  emergency: "Экстренная помощь",
};
interface Questionnaire {
  id: string;
  version: number;
  disclaimer: string;
  questions: {
    id: string;
    label: string;
    variants: Record<string, string> | null;
  }[];
}

export function Survey() {
  const [data, setData] = useState<Questionnaire>();
  const [error, setError] = useState(false);
  const [attempt, retry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setError(false);
    authorized<Questionnaire>(
      "/surveys/interview",
      undefined,
      controller.signal,
    )
      .then(setData)
      .catch(() => {
        if (!controller.signal.aborted) setError(true);
      });
    return () => controller.abort();
  }, [attempt]);
  if (!data)
    return (
      <section>
        <h1>Моя неделя</h1>
        {error ? (
          <>
            <p role="alert">Не удалось загрузить вопросы.</p>
            <button onClick={() => retry(attempt + 1)}>Повторить</button>
          </>
        ) : (
          <p role="status">Загружаем вопросы…</p>
        )}
      </section>
    );
  return <Interview data={data} />;
}

function Interview({ data }: { data: Questionnaire }) {
  const navigate = useNavigate();
  const [step, setStep] = useState(0);
  const [responses, setResponses] = useState<Record<string, string>>({});
  const [ratings, setRatings] = useState<Record<string, number>>({});
  const [focus, setFocus] = useState<Record<string, string>>({});
  const [assisted, setAssisted] = useState(false);
  const [selecting, setSelecting] = useState(false);
  const [notice, setNotice] = useState("");
  const focusRequest = useRef<AbortController | null>(null);
  useEffect(() => () => focusRequest.current?.abort(), []);
  const [summary, setSummary] = useState("");
  const lastDraft = useRef("");
  const [needs, setNeeds] = useState<string[]>([]);
  const [urgent, setUrgent] = useState(false);
  const [consent, setConsent] = useState(false);
  const [external, setExternal] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const submission = useRef({ payload: "", key: "" });
  const q = data.questions[step];
  async function choose(topic: string, scores: Record<string, number>) {
    focusRequest.current?.abort();
    const controller = new AbortController();
    focusRequest.current = controller;
    setSelecting(true);
    setNotice("");
    try {
      const { focus: variant } = await authorized<{ focus: string }>(
        "/surveys/interview/focus",
        { topic, ratings: scores, consent: true },
        controller.signal,
      );
      if (!controller.signal.aborted)
        setFocus((previous) => ({ ...previous, [topic]: variant }));
    } catch {
      if (!controller.signal.aborted) {
        setAssisted(false);
        setNotice("Подбор недоступен. Продолжите с основным вопросом.");
      }
    } finally {
      if (!controller.signal.aborted) setSelecting(false);
    }
  }
  function next(skip = false) {
    const answers = { ...responses },
      scores = { ...ratings };
    if (skip) {
      delete answers[q.id];
      delete scores[q.id];
      setResponses(answers);
      setRatings(scores);
    }
    const nextQuestion = data.questions[step + 1];
    if (!nextQuestion) {
      const draft = Object.entries(answers)
        .filter(([, v]) => v.trim())
        .map(([id, v]) => topics[id] + " — " + v.trim())
        .join("\n")
        .slice(0, 1800);
      if (summary === lastDraft.current) setSummary(draft);
      lastDraft.current = draft;
      if ((scores.expenses ?? 4) > 1)
        setNeeds(needs.filter((n) => n !== "financial"));
    } else if (assisted && !focus[nextQuestion.id])
      void choose(nextQuestion.id, scores);
    setStep(step + 1);
  }
  const answers = Object.fromEntries(
    Object.entries(responses)
      .map(([k, v]) => [k, v.trim()])
      .filter(([, v]) => v),
  );
  const hasContent =
    Object.keys(answers).length + Object.keys(ratings).length + needs.length >
      0 ||
    !!summary.trim() ||
    urgent;
  async function submit() {
    const body = {
      questionnaire_id: data.id,
      version: data.version,
      responses: answers,
      ratings,
      question_focus: focus,
      summary: summary.trim(),
      needs,
      answers: { immediate_threat: urgent },
      consent,
      external_llm_consent: external,
    };
    const payload = JSON.stringify(body);
    if (submission.current.payload !== payload)
      submission.current = { payload, key: crypto.randomUUID() };
    setBusy(true);
    setError("");
    try {
      const result = await authorized<{ id: string }>("/surveys/submissions", {
        ...body,
        idempotency_key: submission.current.key,
      });
      navigate("/assessment/" + encodeURIComponent(result.id), {
        replace: true,
      });
    } catch {
      setError("Не удалось сохранить ответы. Попробуйте ещё раз.");
      setBusy(false);
    }
  }
  return (
    <section className="week">
      <Link to="/profile">← В профиль</Link>
      <h1>Моя неделя</h1>
      <progress
        aria-label="Ход опроса"
        value={step + 1}
        max={data.questions.length + 1}
      />
      {q ? (
        <>
          <p className="eyebrow">
            {step + 1} из {data.questions.length} · {topics[q.id]}
          </p>
          {step === 0 && <p>{data.disclaimer}</p>}
          <label className="check">
            <input
              type="checkbox"
              checked={assisted}
              onChange={(e) => {
                setAssisted(e.target.checked);
                setNotice("");
                if (e.target.checked) void choose(q.id, ratings);
                else {
                  focusRequest.current?.abort();
                  setSelecting(false);
                }
              }}
            />
            Разрешаю Gemini использовать тему и мои оценки для подбора вопроса
          </label>
          {notice && <p role="status">{notice}</p>}
          <h2>{q.variants?.[focus[q.id]] ?? q.label}</h2>
          <label>
            Ваш ответ
            <textarea
              disabled={selecting}
              maxLength={800}
              value={responses[q.id] ?? ""}
              onChange={(e) =>
                setResponses({ ...responses, [q.id]: e.target.value })
              }
            />
          </label>
          {q.id !== "priority" && (
            <fieldset disabled={selecting}>
              <legend>Насколько ситуация отвечала вашим потребностям?</legend>
              <div className="choices">
                {[0, 1, 2, 3, 4].map((n) => (
                  <button
                    key={n}
                    aria-pressed={ratings[q.id] === n}
                    onClick={() => {
                      const scores = { ...ratings };
                      if (scores[q.id] === n) delete scores[q.id];
                      else scores[q.id] = n;
                      setRatings(scores);
                    }}
                  >
                    {n}
                  </button>
                ))}
              </div>
              <small>0 — совсем не отвечала · 4 — полностью</small>
            </fieldset>
          )}
          <button
            className="primary"
            disabled={selecting}
            onClick={() => next()}
          >
            {step === data.questions.length - 1 ? "Проверить ответы" : "Дальше"}
          </button>
          <button
            className="text"
            disabled={selecting}
            onClick={() => next(true)}
          >
            Пропустить вопрос
          </button>
        </>
      ) : (
        <fieldset disabled={busy}>
          <h2>Проверьте ответы</h2>
          <label>
            Моё описание ситуации
            <textarea
              maxLength={1800}
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
            />
          </label>
          {data.questions.map((v, i) => (
            <button className="review" key={v.id} onClick={() => setStep(i)}>
              {topics[v.id]} ·{" "}
              {ratings[v.id] !== undefined ? ratings[v.id] + "/4" : "—"}
              <small>{answers[v.id] ?? "Без ответа"}</small>
            </button>
          ))}
          <h3>Какая помощь нужна?</h3>
          {Object.entries(directions)
            .filter(
              ([k]) =>
                k !== "emergency" &&
                (k !== "financial" || (ratings.expenses ?? 4) <= 1),
            )
            .map(([k, label]) => (
              <label className="check" key={k}>
                <input
                  type="checkbox"
                  checked={needs.includes(k)}
                  onChange={(e) =>
                    setNeeds(
                      e.target.checked
                        ? [...needs, k]
                        : needs.filter((n) => n !== k),
                    )
                  }
                />
                {label}
              </label>
            ))}
          <label className="check">
            <input
              type="checkbox"
              checked={urgent}
              onChange={(e) => setUrgent(e.target.checked)}
            />
            Мне сейчас нужна экстренная помощь
          </label>
          {urgent && (
            <a href="/support/emergency" target="_blank" rel="noreferrer">
              Открыть экстренные номера ↗
            </a>
          )}
          <hr />
          <label className="check">
            <input
              type="checkbox"
              checked={consent}
              onChange={(e) => setConsent(e.target.checked)}
            />
            Подтверждаю описание и оценки. Разрешаю сохранить ответы в SAYA.
          </label>
          <label className="check">
            <input
              type="checkbox"
              checked={external}
              onChange={(e) => setExternal(e.target.checked)}
            />
            Разрешаю передать внешней модели оценки, выбранные направления и
            отметку экстренной помощи для дополнительных рекомендаций. Текст
            ответов остаётся в SAYA.
          </label>
          {error && <p role="alert">{error}</p>}
          <button
            className="primary"
            disabled={!consent || !hasContent || busy}
            onClick={submit}
          >
            {busy ? "Сохраняем…" : "Сохранить результат"}
          </button>
        </fieldset>
      )}
      {step > 0 && (
        <button
          className="text"
          disabled={busy || selecting}
          onClick={() => setStep(step - 1)}
        >
          Назад
        </button>
      )}
    </section>
  );
}
