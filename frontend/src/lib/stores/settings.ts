import { writable } from "svelte/store";
import type { AppSettings } from "@lib/types/common";

export const settings = writable<AppSettings | null>(null);
