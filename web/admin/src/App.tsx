import { useEffect, useState } from "react";
import { Spinner } from "@heroui/react";
import { api } from "./api";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";

export default function App() {
  const [authed, setAuthed] = useState<boolean | null>(null);

  const refresh = () =>
    api
      .me()
      .then((r) => setAuthed(r.authenticated))
      .catch(() => setAuthed(false));

  useEffect(() => {
    refresh();
  }, []);

  if (authed === null) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner label="Loading…" />
      </div>
    );
  }

  if (!authed) return <Login onSuccess={() => setAuthed(true)} />;
  return <Dashboard onLogout={() => setAuthed(false)} />;
}
