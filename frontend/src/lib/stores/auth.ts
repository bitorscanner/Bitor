import { writable } from "svelte/store";
import type { User } from "pocketbase";

export const currentUser = writable<User | null>(null);
