import type { User } from "./types";

const BASE = "/admin/api";

async function req<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(BASE + path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new Error(data.error || `request failed (${res.status})`);
  }
  return data as T;
}

export const api = {
  me: () => req<{ authenticated: boolean }>("/me"),
  login: (password: string) =>
    req<{ ok: boolean }>("/login", {
      method: "POST",
      body: JSON.stringify({ password }),
    }),
  logout: () => req<{ ok: boolean }>("/logout", { method: "POST" }),
  changePassword: (current: string, next: string) =>
    req<{ ok: boolean }>("/change-password", {
      method: "POST",
      body: JSON.stringify({ current, new: next }),
    }),

  listUsers: () => req<User[]>("/users"),
  createUser: (username: string, password: string, email: string) =>
    req<User>("/users", {
      method: "POST",
      body: JSON.stringify({ username, password, email }),
    }),
  updateUser: (
    id: number,
    patch: Partial<{
      password: string;
      email: string;
      downloadRole: boolean;
      adminRole: boolean;
    }>,
  ) =>
    req<User>(`/users/${id}`, {
      method: "PATCH",
      body: JSON.stringify(patch),
    }),
  deleteUser: (id: number) =>
    req<{ ok: boolean }>(`/users/${id}`, { method: "DELETE" }),

  replenishCookies: (id: number, cookies: string) =>
    req<User>(`/users/${id}/apple-token`, {
      method: "POST",
      body: JSON.stringify({ cookies }),
    }),
  recheck: (id: number) =>
    req<User>(`/users/${id}/apple-token/check`, { method: "POST" }),
};
