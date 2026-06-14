import { useState } from "react";
import { Button, Card, CardBody, CardHeader, Input } from "@heroui/react";
import { api } from "../api";

export default function Login({ onSuccess }: { onSuccess: () => void }) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setError("");
    setLoading(true);
    try {
      await api.login(password);
      onSuccess();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="flex-col items-start gap-1 pb-0">
          <h1 className="text-xl font-semibold">APLSonic Admin</h1>
          <p className="text-small text-default-500">
            Enter the admin password to continue.
          </p>
        </CardHeader>
        <CardBody className="gap-4">
          <Input
            type="password"
            label="Password"
            value={password}
            onValueChange={setPassword}
            onKeyDown={(e) => e.key === "Enter" && submit()}
            isInvalid={!!error}
            errorMessage={error}
            autoFocus
          />
          <Button
            color="primary"
            onPress={submit}
            isLoading={loading}
            isDisabled={!password}
          >
            Log in
          </Button>
        </CardBody>
      </Card>
    </div>
  );
}
