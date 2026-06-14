import { useState } from "react";
import {
  Button,
  Input,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Switch,
} from "@heroui/react";
import { api } from "../api";
import type { User } from "../types";

export default function EditUserModal({
  user,
  onClose,
  onSaved,
}: {
  user: User | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState(user?.email ?? "");
  const [downloadRole, setDownloadRole] = useState(user?.downloadRole ?? true);
  const [adminRole, setAdminRole] = useState(user?.adminRole ?? false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!user) return;
    setError("");
    setLoading(true);
    try {
      await api.updateUser(user.id, {
        ...(password ? { password } : {}),
        email,
        downloadRole,
        adminRole,
      });
      onSaved();
      onClose();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={!!user} onClose={onClose} placement="center">
      <ModalContent>
        <ModalHeader>Edit {user?.username}</ModalHeader>
        <ModalBody>
          <Input
            label="New password"
            type="password"
            placeholder="leave blank to keep current"
            value={password}
            onValueChange={setPassword}
          />
          <Input label="Email" value={email} onValueChange={setEmail} />
          <Switch isSelected={downloadRole} onValueChange={setDownloadRole}>
            Download role
          </Switch>
          <Switch isSelected={adminRole} onValueChange={setAdminRole}>
            Admin role
          </Switch>
          {error && <p className="text-small text-danger">{error}</p>}
        </ModalBody>
        <ModalFooter>
          <Button variant="light" onPress={onClose}>
            Cancel
          </Button>
          <Button color="primary" onPress={submit} isLoading={loading}>
            Save
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
