import type { Writable } from "svelte/store";

export function blurActiveElement(): void {
	const el = document.activeElement;
	if (el instanceof HTMLElement) {
		el.blur();
	}
}

export function closeDialog(open: Writable<boolean>): void {
	blurActiveElement();
	open.set(false);
}
