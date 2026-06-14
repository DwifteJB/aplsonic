import { useEffect, useState } from "react";
import {
  Button,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableColumn,
  TableHeader,
  TableRow,
  Tooltip,
} from "@heroui/react";
import { api } from "../api";
import type { User } from "../types";
import { StatusChip } from "../lib/status";
import CreateUserModal from "../components/CreateUserModal";
import EditUserModal from "../components/EditUserModal";
import ReplenishModal from "../components/ReplenishModal";
import SettingsModal from "../components/SettingsModal";

export default function Dashboard({ onLogout }: { onLogout: () => void }) {
  const [users, setUsers] = useState<User[] | null>(null);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [replenishUser, setReplenishUser] = useState<User | null>(null);
  const [rechecking, setRechecking] = useState<number | null>(null);

  const load = () =>
    api
      .listUsers()
      .then(setUsers)
      .catch((e) => setError((e as Error).message));

  useEffect(() => {
    load();
  }, []);

  const replaceUser = (u: User) =>
    setUsers((prev) => (prev ? prev.map((x) => (x.id === u.id ? u : x)) : prev));

  const recheck = async (u: User) => {
    setRechecking(u.id);
    try {
      replaceUser(await api.recheck(u.id));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setRechecking(null);
    }
  };

  const remove = async (u: User) => {
    if (!confirm(`Delete ${u.username}?`)) return;
    try {
      await api.deleteUser(u.id);
      load();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const logout = async () => {
    await api.logout().catch(() => {});
    onLogout();
  };

  return (
    <div className="mx-auto max-w-5xl p-6">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">APLSonic Admin</h1>
          <p className="text-small text-default-500">
            Accounts and Apple Music tokens
          </p>
        </div>
        <div className="flex gap-2">
          <Button color="primary" onPress={() => setCreating(true)}>
            Add account
          </Button>
          <Button variant="flat" onPress={() => setSettingsOpen(true)}>
            Settings
          </Button>
          <Button variant="light" onPress={logout}>
            Log out
          </Button>
        </div>
      </header>

      {error && <p className="mb-4 text-small text-danger">{error}</p>}

      {!users ? (
        <div className="flex justify-center py-16">
          <Spinner label="Loading accounts…" />
        </div>
      ) : (
        <Table aria-label="accounts">
          <TableHeader>
            <TableColumn>USER</TableColumn>
            <TableColumn>APPLE TOKEN</TableColumn>
            <TableColumn>LAST CHECKED</TableColumn>
            <TableColumn align="end">ACTIONS</TableColumn>
          </TableHeader>
          <TableBody emptyContent="No accounts yet. Add one to get started.">
            {users.map((u) => (
              <TableRow key={u.id}>
                <TableCell>
                  <div className="font-medium">{u.username}</div>
                  <div className="text-tiny text-default-400">{u.email}</div>
                </TableCell>
                <TableCell>
                  <Tooltip
                    content={u.appleTokenLastError || "no recent errors"}
                    isDisabled={!u.appleTokenLastError}
                  >
                    <span>
                      <StatusChip user={u} />
                    </span>
                  </Tooltip>
                </TableCell>
                <TableCell className="text-tiny text-default-400">
                  {u.appleTokenLastCheckedAt
                    ? new Date(u.appleTokenLastCheckedAt).toLocaleString()
                    : "unknown"}
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-1">
                    <Button
                      size="sm"
                      color="primary"
                      variant="flat"
                      onPress={() => setReplenishUser(u)}
                    >
                      Replenish
                    </Button>
                    <Button
                      size="sm"
                      variant="light"
                      isLoading={rechecking === u.id}
                      onPress={() => recheck(u)}
                    >
                      Recheck
                    </Button>
                    <Button size="sm" variant="light" onPress={() => setEditUser(u)}>
                      Edit
                    </Button>
                    <Button
                      size="sm"
                      color="danger"
                      variant="light"
                      onPress={() => remove(u)}
                    >
                      Delete
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CreateUserModal
        isOpen={creating}
        onClose={() => setCreating(false)}
        onCreated={load}
      />
      <SettingsModal isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <EditUserModal
        user={editUser}
        onClose={() => setEditUser(null)}
        onSaved={load}
      />
      <ReplenishModal
        user={replenishUser}
        onClose={() => setReplenishUser(null)}
        onUpdated={replaceUser}
      />
    </div>
  );
}
