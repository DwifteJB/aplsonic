import { useState } from "react";
import {
  Button,
  Link,
  Modal,
  ModalBody,
  ModalContent,
  ModalHeader,
  Textarea,
} from "@heroui/react";
import { api } from "../api";
import type { User } from "../types";

export default function ReplenishModal({
  user,
  onClose,
  onUpdated,
}: {
  user: User | null;
  onClose: () => void;
  onUpdated: (u: User) => void;
}) {
  const [cookies, setCookies] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const doCookies = async () => {
    if (!user) return;
    setError("");
    setSuccess("");
    setBusy(true);
    try {
      const u = await api.replenishCookies(user.id, cookies);
      onUpdated(u);
      setCookies("");
      if (u.appleTokenStatus === "expired") {
        setError("Saved, but Apple rejected the token (expired/invalid).");
      } else {
        setSuccess("Saved token!");
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const onFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    f.text().then(setCookies);
  };

  return (
    <Modal isOpen={!!user} onClose={onClose} placement="center" size="2xl">
      <ModalContent>
        <ModalHeader className="flex-col items-start">
          Replenish Apple token
          <span className="text-small font-normal text-default-500">
            for {user?.username}
          </span>
        </ModalHeader>
        <ModalBody className="pb-6">
          <ol className="list-decimal pl-5 text-small text-default-500 space-y-1">
            <li>
              Install{" "}
              <Link
                isExternal
                size="sm"
                href="https://chromewebstore.google.com/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc"
              >
                Get cookies.txt LOCALLY
              </Link>{" "}
              (Chrome) or{" "}
              <Link
                isExternal
                size="sm"
                href="https://addons.mozilla.org/en-US/firefox/addon/get-cookies-txt-locally/"
              >
                the Firefox version
              </Link>
              .
            </li>
            <li>
              Open{" "}
              <Link isExternal size="sm" href="https://music.apple.com">
                music.apple.com
              </Link>{" "}
              while logged in, click the extension, and Export.
            </li>
            <li>Drop the file below (or paste its contents).</li>
          </ol>
          <input
            type="file"
            accept=".txt"
            onChange={onFile}
            className="text-small"
          />
          <Textarea
            label="cookies.txt"
            placeholder="# Netscape HTTP Cookie File …"
            minRows={5}
            value={cookies}
            onValueChange={setCookies}
          />
          <Button
            color="primary"
            onPress={doCookies}
            isLoading={busy}
            isDisabled={!cookies.trim()}
          >
            Save &amp; verify
          </Button>

          {error && <p className="text-small text-danger">{error}</p>}
          {success && <p className="text-small text-success">{success}</p>}
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}
