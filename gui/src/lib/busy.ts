import { derived, writable } from "svelte/store";

export const busyMessage = writable<string | null>(null);

const messageStack: string[] = [];

export const isBusy = derived(busyMessage, ($message) => $message !== null);

export async function withBusy<T>(message: string, fn: () => Promise<T>): Promise<T> {
	messageStack.push(message);
	busyMessage.set(message);
	try {
		return await fn();
	} finally {
		messageStack.pop();
		busyMessage.set(messageStack.at(-1) ?? null);
	}
}
