export type AppleTokenStatus =
  | "ok"
  | "expiring"
  | "expired"
  | "unknown"
  | "missing";

export interface User {
  id: number;
  username: string;
  email: string;
  adminRole: boolean;
  downloadRole: boolean;
  hasAppleToken: boolean;
  appleTokenStatus: AppleTokenStatus;
  appleTokenExpiresAt: string | null;
  appleTokenLastCheckedAt: string | null;
  appleTokenLastError: string;
}
