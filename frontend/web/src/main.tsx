import { Survey } from "./survey";
import { Assessment, LatestAssessment } from "./assessment";
import { useEffect, useState, type FormEvent } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Link, Navigate, Route, Routes } from "react-router-dom";
import { api, ApiError, type Profile } from "./api";
import "./style.css";

function Cover() {
  return (
    <aside className="cover" aria-label="Saya — пространство поддержки">
      <div className="brand">
        <span aria-hidden="true">✳</span> saya
      </div>
      <div className="landscape" aria-hidden="true">
        <i className="sun" />
        <i className="ridge far" />
        <i className="ridge middle" />
        <i className="ridge near" />
      </div>
      <div className="caption">
        Пространство
        <br />
        <em>поддержки.</em>
      </div>
    </aside>
  );
}

function message(error: unknown, code = false) {
  if (error instanceof ApiError) {
    if (error.status === 401)
      return code ? "Код неверен или истёк." : "Проверьте почту и пароль.";
    if (error.status === 429)
      return "Слишком много попыток. Попробуйте чуть позже.";
    if (error.status === 409)
      return "Эта почта уже используется. Попробуйте войти.";
    if (error.status === 400) return "Проверьте введённые данные.";
  }
  return "Не удалось выполнить запрос. Проверьте соединение и попробуйте ещё раз.";
}

function AuthForm({
  register,
  onSession,
}: {
  register: boolean;
  onSession: (user: Profile) => void;
}) {
  const [challenge, setChallenge] = useState("");
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [show, setShow] = useState(false);
  const [resendAt, setResendAt] = useState(0);
  const [now, setNow] = useState(Date.now);
  const [registration, setRegistration] = useState<{
    email: string;
    password: string;
    full_name: string;
    phone: string;
  } | null>(null);
  const remaining = Math.max(0, Math.ceil((resendAt - now) / 1000));
  useEffect(() => {
    if (!challenge) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [challenge]);
  async function send(data: NonNullable<typeof registration>) {
    const result = await api.register(data);
    setChallenge(result.registration_id);
    setRegistration(data);
    setNow(Date.now());
    setResendAt(Date.now() + 60_000);
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const field = (key: string) => String(form.get(key) ?? "");
    setBusy(true);
    setError("");
    try {
      if (challenge) onSession(await api.confirm(challenge, field("code")));
      else if (register)
        await send({
          email: email.trim(),
          password: field("password"),
          full_name: field("name").trim(),
          phone: field("phone").trim(),
        });
      else onSession(await api.login(email.trim(), field("password")));
    } catch (err) {
      setError(message(err, Boolean(challenge)));
    } finally {
      setBusy(false);
    }
  }
  async function resend() {
    if (!registration || remaining || busy) return;
    setBusy(true);
    setError("");
    try {
      await send(registration);
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section>
      <h1>
        {challenge
          ? "Проверьте почту"
          : register
            ? "Регистрация"
            : "С возвращением"}
      </h1>
      <p className="intro">
        {challenge
          ? "Введите шестизначный код из письма на " + email
          : register
            ? "Создайте свой личный кабинет Saya."
            : "Войдите в свой личный кабинет Saya."}
      </p>
      <form onSubmit={submit} key={challenge || "credentials"}>
        {challenge ? (
          <label>
            Код из письма
            <input
              name="code"
              autoComplete="one-time-code"
              inputMode="numeric"
              pattern="[0-9]{6}"
              maxLength={6}
              required
              autoFocus
            />
          </label>
        ) : (
          <>
            {register && (
              <>
                <label>
                  Полное имя
                  <input
                    name="name"
                    autoComplete="name"
                    required
                    maxLength={180}
                  />
                </label>
                <label>
                  Телефон
                  <input
                    type="tel"
                    name="phone"
                    autoComplete="tel"
                    required
                    placeholder="+7 700 000 00 00"
                  />
                </label>
              </>
            )}
            <label>
              Электронная почта
              <input
                type="email"
                autoComplete="email"
                name="email"
                placeholder="name@example.com"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
            <label>
              Пароль
              <span className="password">
                <input
                  type={show ? "text" : "password"}
                  name="password"
                  autoComplete={register ? "new-password" : "current-password"}
                  minLength={register ? 12 : undefined}
                  required
                  placeholder={register ? "От 12 символов" : "Введите пароль"}
                />
                <button
                  type="button"
                  aria-label={show ? "Скрыть пароль" : "Показать пароль"}
                  aria-pressed={show}
                  onClick={() => setShow(!show)}
                >
                  <svg
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    aria-hidden="true"
                  >
                    <path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12Z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                </button>
              </span>
            </label>
          </>
        )}
        {error && <p role="alert">{error}</p>}
        <button className="primary" disabled={busy}>
          {busy
            ? "Подождите…"
            : challenge
              ? "Подтвердить почту"
              : register
                ? "Создать аккаунт"
                : "Войти"}{" "}
          <span aria-hidden="true">↗</span>
        </button>
      </form>
      {challenge && (
        <button
          className="text"
          disabled={busy || remaining > 0}
          onClick={resend}
        >
          Отправить код ещё раз{remaining > 0 && " · " + remaining + " с"}
        </button>
      )}
      <p className="foot">
        {register ? "Уже есть аккаунт?" : "Впервые в Saya?"}{" "}
        <Link to={register ? "/login" : "/register"}>
          {register ? "Войти" : "Зарегистрироваться"}
        </Link>
      </p>
    </section>
  );
}

function App() {
  const [user, setUser] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const [leaving, setLeaving] = useState(false);
  async function restore() {
    setLoading(true);
    setFailed(false);
    try {
      setUser(await api.restore());
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void restore();
  }, []);
  async function logout() {
    setLeaving(true);
    setLogoutError("");
    try {
      await api.logout();
      setUser(null);
    } catch (err) {
      setLogoutError(message(err));
    } finally {
      setLeaving(false);
    }
  }
  return (
    <main>
      <Cover />
      <div className="content">
        {loading ? (
          <p role="status">Открываем кабинет…</p>
        ) : failed ? (
          <section>
            <h1>Нет соединения</h1>
            <p>Не удалось проверить сессию.</p>
            <button className="primary" onClick={restore}>
              Повторить
            </button>
          </section>
        ) : (
          <Routes>
            <Route
              path="/survey"
              element={user ? <Survey /> : <Navigate to="/login" replace />}
            />
            <Route
              path="/assessment/:id"
              element={
                user ? (
                  <Assessment owner={user.id} />
                ) : (
                  <Navigate to="/login" replace />
                )
              }
            />
            {["login", "register"].map((path) => (
              <Route
                key={path}
                path={"/" + path}
                element={
                  user ? (
                    <Navigate to="/profile" replace />
                  ) : (
                    <AuthForm
                      key={path}
                      register={path === "register"}
                      onSession={setUser}
                    />
                  )
                }
              />
            ))}
            <Route
              path="/profile"
              element={
                user ? (
                  <section>
                    <h1>Ваш профиль</h1>
                    <dl>
                      <dt>Электронная почта</dt>
                      <dd>{user.email}</dd>
                      <dt>Роль</dt>
                      <dd>
                        {user.role === "USER" ? "Пользователь" : user.role}
                      </dd>
                    </dl>
                    <p>
                      <Link to="/survey">Пройти опрос «Моя неделя» →</Link>
                    </p>
                    {logoutError && <p role="alert">{logoutError}</p>}
                    <LatestAssessment />
                    <button
                      className="primary"
                      onClick={logout}
                      disabled={leaving}
                    >
                      Выйти
                    </button>
                  </section>
                ) : (
                  <Navigate to="/login" replace />
                )
              }
            />
            <Route
              path="*"
              element={<Navigate to={user ? "/profile" : "/login"} replace />}
            />
          </Routes>
        )}
      </div>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <BrowserRouter>
    <App />
  </BrowserRouter>,
);
