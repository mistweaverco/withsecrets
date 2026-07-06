import { get, writable } from "svelte/store";
import { getSecret } from "$lib/api";

const values = writable<Record<string, string>>({});

function cacheKey(env: string, envVar: string): string {
	return `${env}\0${envVar}`;
}

export function peekSecretValue(env: string, envVar: string): string | undefined {
	return get(values)[cacheKey(env, envVar)];
}

export function setSecretValue(env: string, envVar: string, value: string): void {
	values.update((cache) => ({ ...cache, [cacheKey(env, envVar)]: value }));
}

export function deleteSecretValue(env: string, envVar: string): void {
	const k = cacheKey(env, envVar);
	values.update((cache) => {
		if (!(k in cache)) return cache;
		const next = { ...cache };
		delete next[k];
		return next;
	});
}

export function cacheSecretValues(env: string, rows: { envVar: string; value: string }[]): void {
	values.update((cache) => {
		const next = { ...cache };
		for (const row of rows) {
			next[cacheKey(env, row.envVar)] = row.value;
		}
		return next;
	});
}

export async function resolveSecretValue(env: string, envVar: string): Promise<string> {
	const cached = peekSecretValue(env, envVar);
	if (cached !== undefined) return cached;

	const value = await getSecret(env, envVar);
	setSecretValue(env, envVar, value);
	return value;
}
