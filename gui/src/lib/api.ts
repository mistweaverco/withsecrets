export type EnvironmentSummary = {
	name: string;
	provider: string;
	project: string;
};

export type SecretRow = {
	envVar: string;
	value: string;
	maskedValue: string;
	refKind: string;
	ref: string;
	provider: string;
	project: string;
};

export type CreateOptions = {
	provider: string;
	replication: string;
	locations: string[];
	allLocations?: string[];
	selectedLocations?: string[];
	supportsReplication: boolean;
};

class ApiError extends Error {
	constructor(message: string) {
		super(message);
		this.name = "ApiError";
	}
}

const CSRF_HEADER = "X-WS-GUI-Token";

let authToken: string | null = null;
let authTokenPromise: Promise<string> | null = null;

async function ensureAuthToken(): Promise<string> {
	if (authToken) return authToken;
	if (!authTokenPromise) {
		authTokenPromise = fetchAuthToken();
	}
	try {
		authToken = await authTokenPromise;
		return authToken;
	} catch (err) {
		authTokenPromise = null;
		throw err;
	}
}

async function fetchAuthToken(): Promise<string> {
	const res = await fetch("/api/auth/token");
	const text = await res.text();
	if (!res.ok) {
		throw new ApiError(text || "Failed to obtain API token");
	}
	const body = JSON.parse(text) as { token?: string };
	if (!body.token) {
		throw new ApiError("Failed to obtain API token");
	}
	return body.token;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const token = await ensureAuthToken();
	const res = await fetch(path, {
		headers: {
			"Content-Type": "application/json",
			[CSRF_HEADER]: token,
			...(init?.headers ?? {})
		},
		...init
	});
	const text = await res.text();
	if (!res.ok) {
		let message = res.statusText;
		if (text) {
			try {
				const body = JSON.parse(text) as { error?: string };
				if (body.error) message = body.error;
			} catch {
				message = text;
			}
		}
		throw new ApiError(message);
	}
	if (!text) {
		return undefined as T;
	}
	return JSON.parse(text) as T;
}

export async function listEnvironments(): Promise<EnvironmentSummary[]> {
	const data = await request<{ environments: EnvironmentSummary[] }>("/api/environments");
	return data.environments;
}

export async function listSecrets(env: string): Promise<SecretRow[]> {
	const data = await request<{ secrets: SecretRow[] }>(
		`/api/environments/${encodeURIComponent(env)}/secrets`
	);
	return data.secrets;
}

export async function getSecret(env: string, envVar: string): Promise<string> {
	const data = await request<{ value: string }>(
		`/api/environments/${encodeURIComponent(env)}/secrets/${encodeURIComponent(envVar)}`
	);
	return data.value;
}

export async function getCreateOptions(env: string): Promise<CreateOptions> {
	return request<CreateOptions>(`/api/environments/${encodeURIComponent(env)}/create-options`);
}

export async function createSecret(
	env: string,
	body: {
		envVar: string;
		secretKey: string;
		value: string;
		description?: string;
		replication?: string;
		locations?: string[];
	}
): Promise<void> {
	await request(`/api/environments/${encodeURIComponent(env)}/secrets`, {
		method: "POST",
		body: JSON.stringify(body)
	});
}

export async function updateSecret(env: string, envVar: string, value: string): Promise<void> {
	await request(`/api/environments/${encodeURIComponent(env)}/secrets/${encodeURIComponent(envVar)}`, {
		method: "PUT",
		body: JSON.stringify({ value })
	});
}

export async function deleteSecret(env: string, envVar: string): Promise<void> {
	await request(`/api/environments/${encodeURIComponent(env)}/secrets/${encodeURIComponent(envVar)}`, {
		method: "DELETE"
	});
}
