import { writable } from "svelte/store";

export type ToastType = "success" | "error" | "warning" | "info";

export interface Toast {
  id?: string;
  message: string;
  type: ToastType;
  duration?: number;
}

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);

  function addToast(toast: Toast) {
    const id = Math.random().toString(36).substring(2);
    const duration = toast.duration || 5000;

    update((toasts) => [...toasts, { ...toast, id }]);

    setTimeout(() => {
      removeToast(id);
    }, duration);
  }

  function removeToast(id: string) {
    update((toasts) => toasts.filter((t) => t.id !== id));
  }

  return {
    subscribe,
    add: addToast,
    remove: removeToast,
  };
}

export const toasts = createToastStore();
export const addToast = toasts.add;
