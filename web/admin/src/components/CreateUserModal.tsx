import { useState } from "react";
import {
  Button,
  Input,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
} from "@heroui/react";
import { api } from "../api";

export default function CreateUserModal({
  isOpen,
  onClose,
  onCreated,
}: {
  isOpen: boolean;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setError("");
    setLoading(true);
    try {
      await api.createUser(username, password, email);
      setUsername("");
      setPassword("");
      setEmail("");
      onCreated();
      onClose();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} placement="center">
      <ModalContent>
        <ModalHeader>New account</ModalHeader>
        <ModalBody>
          <p className="text-small text-default-500">
            Creates a Subsonic login. Attach an Apple token afterwards with
            “Replenish”.
          </p>
          <Input label="Username" value={username} onValueChange={setUsername} autoFocus />
          <Input
            label="Password"
            type="password"
            value={password}
            onValueChange={setPassword}
          />
          <Input label="Email (optional)" value={email} onValueChange={setEmail} />
          {error && <p className="text-small text-danger">{error}</p>}
        </ModalBody>
        <ModalFooter>
          <Button variant="light" onPress={onClose}>
            Cancel
          </Button>
          <Button
            color="primary"
            onPress={submit}
            isLoading={loading}
            isDisabled={!username || !password}
          >
            Create
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
