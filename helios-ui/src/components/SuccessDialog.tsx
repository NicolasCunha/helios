import { CheckCircle } from 'lucide-react';
import Modal from './Modal';

interface SuccessDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  message: string;
  details?: string;
}

export default function SuccessDialog({
  isOpen,
  onClose,
  title,
  message,
  details,
}: SuccessDialogProps) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      footer={
        <button
          onClick={onClose}
          className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors"
        >
          OK
        </button>
      }
    >
      <div className="flex gap-4">
        <div className="flex-shrink-0">
          <CheckCircle className="w-6 h-6 text-green-500" />
        </div>
        <div className="flex-1">
          <p className="text-gray-200 break-words whitespace-pre-wrap">{message}</p>
          {details && (
            <div className="mt-4 p-3 bg-gray-900/50 rounded border border-gray-700">
              <p className="text-sm text-gray-400 font-mono whitespace-pre-wrap break-words">
                {details}
              </p>
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
}
