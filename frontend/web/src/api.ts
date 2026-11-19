export interface Profile {
  id: string;
  email: string;
  role: string;
}
interface Session {
  access_token: string;
  csrf_token: string;
  user: Profile;
}

export class ApiError extends Error {
  constructor(readonly status: number) {
    super(`HTTP ${status}`);
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch("/api/v1" + path, {
    credentials: "include",
    ...options,
  });
  if (!response.ok) throw new ApiError(response.status);
  return response.status === 204 ? (undefined as T) : response.json();
}

function post<T>(path: string, data: object) {
  return request<T>(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

let access = "";
async function profile(session: Session) {
  access = session.access_token;
  return request<Profile>("/users/me", {
    headers: { Authorization: "Bearer " + access },
  });
}

async function csrfPost<T>(action: string) {
  const { csrf_token } = await request<{ csrf_token: string }>("/auth/csrf");
  return request<T>("/auth/" + action, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf_token },
  });
}

export const api = {
  async restore() {
    try {
      return await profile(await csrfPost<Session>("refresh"));
    } catch (error) {
      if (error instanceof ApiError && [401, 403].includes(error.status)) {
        access = "";
        return null;
      }
      throw error;
    }
  },
  async login(email: string, password: string) {
    return profile(await post<Session>("/auth/login", { email, password }));
  },
  register(data: {
    email: string;
    password: string;
    full_name: string;
    phone: string;
  }) {
    return post<{ registration_id: string }>("/auth/register", data);
  },
  async confirm(registration_id: string, code: string) {
    return profile(
      await post<Session>("/auth/register/confirm", { registration_id, code }),
    );
  },
  async logout() {
    try {
      await csrfPost<void>("logout");
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) throw error;
    }
    access = "";
  },
};
