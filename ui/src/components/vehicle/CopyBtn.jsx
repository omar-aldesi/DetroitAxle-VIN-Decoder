import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { useToast } from "../../contexts/ToastContext";
import { copyText } from "../../utils/clipboard";

export function CopyBtn({ text, label = "Copied" }) {
  const toast = useToast();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await copyText(text);
      setCopied(true);
      toast(`${label} copied`, "info");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast("Could not copy to clipboard", "error");
    }
  };

  return (
    <button
      onClick={copy}
      title="Copy to clipboard"
      className="text-txt-muted hover:text-accent transition-colors"
    >
      {copied ? (
        <Check className="w-3.5 h-3.5 text-success" />
      ) : (
        <Copy className="w-3.5 h-3.5" />
      )}
    </button>
  );
}
