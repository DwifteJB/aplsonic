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

export default function SettingsModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setError("");
    setLoading(true);
    try {
      await api.changePassword(current, next);
      setDone(true);
      setCurrent("");
      setNext("");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} placement="center">
      <ModalContent>
        <ModalHeader>Admin password</ModalHeader>
        <ModalBody>
          <Input
            label="Current password"
            type="password"
            value={current}
            onValueChange={setCurrent}
          />
          <Input
            label="New password"
            type="password"
            value={next}
            onValueChange={setNext}
          />
          {error && <p className="text-small text-danger">{error}</p>}
          {done && <p className="text-small text-success">Password updated.</p>}
        </ModalBody>
        <ModalFooter>
          <Button variant="light" onPress={onClose}>
            Close
          </Button>
          <Button
            color="primary"
            onPress={submit}
            isLoading={loading}
            isDisabled={!current || next.length < 6}
          >
            Update
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
