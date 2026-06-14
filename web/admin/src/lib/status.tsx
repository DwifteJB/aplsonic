import { Chip } from "@heroui/react";
import type { AppleTokenStatus, User } from "../types";

function daysUntil(iso: string | null): number | null {
  if (!iso) return null;
  const ms = new Date(iso).getTime() - Date.now();
  return Math.round(ms / (24 * 60 * 60 * 1000));
}

const COLOR: Record<AppleTokenStatus, "success" | "danger" | "warning" | "default"> = {
  ok: "success",
  expiring: "warning",
  expired: "danger",
  unknown: "warning",
  missing: "default",
};

export function StatusChip({ user }: { user: User }) {
  const days = daysUntil(user.appleTokenExpiresAt);
  let label: string;

  switch (user.appleTokenStatus) {
    case "missing":
      label = "no token";
      break;
    case "expired":
      label = "expired";
      break;
    case "expiring":
      label = days !== null ? `expires in ${days}d` : "expiring soon";
      break;
    case "ok":
      label = days !== null ? `expires in ${days}d` : "active";
      break;
    default:
      label =
        days !== null && days >= 0 ? `unverified · ${days}d left` : "unverified";
  }

  return (
    <Chip color={COLOR[user.appleTokenStatus]} variant="flat" size="sm">
      {label}
    </Chip>
  );
}
