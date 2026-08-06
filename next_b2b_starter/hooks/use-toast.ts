// hooks/use-toast.ts
// Simple toast implementation for now

interface ToastProps {
  title?: string;
  description?: string;
  variant?: "default" | "destructive";
}

export function useToast() {
  const toast = ({ title, description, variant }: ToastProps) => {
    // For now, just use console and alert
    // This can be replaced with a proper toast library later
    const message = `${title}${description ? `: ${description}` : ''}`;

    if (variant === "destructive") {
      console.error(message);
      // In production, this would show a red toast
    } else {
      console.log(message);
      // In production, this would show a success/info toast
    }
  };

  return { toast };
}