import { XCircle } from 'lucide-react';
import Modal from './Modal';

interface ErrorDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  error: string;
  details?: string;
}

export default function ErrorDialog({
  isOpen,
  onClose,
  title,
  error,
  details,
}: ErrorDialogProps) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      footer={
        <button
          onClick={onClose}
          className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
        >
          Close
        </button>
      }
    >
      <div className="space-y-4">
        <div className="flex items-start gap-4">
          <div className="p-3 rounded-lg bg-red-500/20 flex-shrink-0">
            <XCircle className="w-6 h-6 text-red-400" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-gray-300 font-medium break-words">{error}</p>
          </div>
        </div>
        
        {details && (
          <div className="mt-4 p-4 bg-gray-900/50 rounded-lg border border-gray-700">
            <p className="text-xs font-mono text-gray-400 whitespace-pre-wrap break-words">{details}</p>
          </div>
        )}
      </div>
    </Modal>
  );
}
