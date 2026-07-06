<script lang="ts">
	import { createDialog, melt } from "@melt-ui/svelte";
	import { onMount } from "svelte";
	import {
		createSecret,
		deleteSecret,
		getCreateOptions,
		listEnvironments,
		listSecrets,
		updateSecret,
		type CreateOptions,
		type EnvironmentSummary,
		type SecretRow
	} from "$lib/api";
	import { closeDialog } from "$lib/dialog";
	import { isBusy, withBusy } from "$lib/busy";
	import LoadingOverlay from "$lib/LoadingOverlay.svelte";
	import {
		cacheSecretValues,
		deleteSecretValue,
		peekSecretValue,
		setSecretValue
	} from "$lib/secretCache";

	let environments = $state<EnvironmentSummary[]>([]);
	let selectedEnv = $state<string | null>(null);
	let secrets = $state<SecretRow[]>([]);
	let filter = $state("");
	let masked = $state(true);
	let secretsLoaded = $state(false);
	let errorMessage = $state("");

	let viewValue = $state("");
	let viewEnvVar = $state("");
	let viewCopied = $state(false);

	let editValue = $state("");
	let editEnvVar = $state("");

	let createEnvVar = $state("");
	let createSecretKey = $state("");
	let createValue = $state("");
	let createDesc = $state("");
	let createReplication = $state("global");
	let createLocations = $state<string[]>([]);
	let createOptions = $state<CreateOptions | null>(null);

	let deleteEnvVar = $state("");
	let deleteRef = $state("");

	const viewDialog = createDialog();
	const editDialog = createDialog();
	const createDialogState = createDialog();
	const deleteDialog = createDialog();
	const errorDialog = createDialog();

	const {
		elements: { overlay: viewOverlay, content: viewContent, close: viewClose },
		states: { open: viewOpen }
	} = viewDialog;

	const {
		elements: { overlay: editOverlay, content: editContent, close: editClose },
		states: { open: editOpen }
	} = editDialog;

	const {
		elements: { overlay: createOverlay, content: createContent, close: createClose },
		states: { open: createOpen }
	} = createDialogState;

	const {
		elements: { overlay: deleteOverlay, content: deleteContent, close: deleteClose },
		states: { open: deleteOpen }
	} = deleteDialog;

	const {
		elements: { overlay: errorOverlay, content: errorContent, close: errorClose },
		states: { open: errorOpen }
	} = errorDialog;

	function showError(message: string) {
		errorMessage = message;
		errorOpen.set(true);
	}

	const filteredSecrets = $derived(
		secrets.filter((row) => {
			const q = filter.trim().toLowerCase();
			if (!q) return true;
			return (
				row.envVar.toLowerCase().includes(q) ||
				row.ref.toLowerCase().includes(q) ||
				row.refKind.toLowerCase().includes(q)
			);
		})
	);

	async function loadEnvironments() {
		try {
			await withBusy("Loading environments…", async () => {
				environments = await listEnvironments();
				if (!selectedEnv && environments.length > 0) {
					selectedEnv = environments[0].name;
				}
			});
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to load environments");
		}
	}

	async function loadSecrets() {
		if (!selectedEnv) return;
		try {
			await withBusy("Loading secrets…", async () => {
				secrets = await listSecrets(selectedEnv);
				cacheSecretValues(selectedEnv, secrets);
				secretsLoaded = true;
			});
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to load secrets");
		}
	}

	async function selectEnvironment(name: string) {
		selectedEnv = name;
		secretsLoaded = false;
		await loadSecrets();
	}

	async function openView(row: SecretRow) {
		if (!selectedEnv) return;
		viewEnvVar = row.envVar;
		viewValue = peekSecretValue(selectedEnv, row.envVar) ?? row.value;
		viewCopied = false;
		viewOpen.set(true);
	}

	async function copyViewValue() {
		try {
			await navigator.clipboard.writeText(viewValue);
			viewCopied = true;
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to copy to clipboard");
		}
	}

	async function openEdit(row: SecretRow) {
		if (!selectedEnv) return;
		if (row.refKind !== "secret-key") {
			showError("Edit is only supported for secret-key mappings");
			return;
		}
		editEnvVar = row.envVar;
		editValue = peekSecretValue(selectedEnv, row.envVar) ?? row.value;
		editOpen.set(true);
	}

	async function submitEdit() {
		if (!selectedEnv) return;
		try {
			await withBusy("Saving secret…", async () => {
				await updateSecret(selectedEnv, editEnvVar, editValue);
				setSecretValue(selectedEnv, editEnvVar, editValue);
				closeDialog(editOpen);
				await loadSecrets();
			});
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to update secret");
		}
	}

	async function openCreate() {
		if (!selectedEnv) return;
		try {
			await withBusy("Loading create options…", async () => {
				createOptions = await getCreateOptions(selectedEnv);
				createEnvVar = "";
				createSecretKey = "";
				createValue = "";
				createDesc = "";
				createReplication = createOptions.replication;
				createLocations = [...(createOptions.selectedLocations ?? [])];
				createOpen.set(true);
			});
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to load create options");
		}
	}

	async function submitCreate() {
		if (!selectedEnv) return;
		try {
			await withBusy("Creating secret…", async () => {
				await createSecret(selectedEnv, {
					envVar: createEnvVar,
					secretKey: createSecretKey,
					value: createValue,
					description: createDesc,
					replication: createReplication,
					locations: createReplication === "user-managed" ? createLocations : []
				});
				setSecretValue(selectedEnv, createEnvVar, createValue);
				closeDialog(createOpen);
				await loadSecrets();
			});
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to create secret");
		}
	}

	function openDelete(row: SecretRow) {
		if (row.refKind !== "secret-key") {
			showError("Delete is only supported for secret-key mappings");
			return;
		}
		deleteEnvVar = row.envVar;
		deleteRef = row.ref;
		deleteOpen.set(true);
	}

	async function submitDelete() {
		if (!selectedEnv) return;
		try {
			await withBusy("Deleting secret…", async () => {
				await deleteSecret(selectedEnv, deleteEnvVar);
				deleteSecretValue(selectedEnv, deleteEnvVar);
				closeDialog(deleteOpen);
				await loadSecrets();
			});
		} catch (e) {
			showError(e instanceof Error ? e.message : "Failed to delete secret");
		}
	}

	function toggleLocation(loc: string) {
		if (createLocations.includes(loc)) {
			createLocations = createLocations.filter((l) => l !== loc);
		} else {
			createLocations = [...createLocations, loc];
		}
	}

	function toggleMask() {
		masked = !masked;
	}

	onMount(async () => {
		await loadEnvironments();
		if (selectedEnv) await loadSecrets();
	});
</script>

<LoadingOverlay />

<div class="grid gap-6 lg:grid-cols-[240px_1fr]">
	<aside class="card p-4">
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-[#9aa0ae]">Environments</h2>
		<ul class="space-y-1">
			{#each environments as env}
				<li>
					<button
						type="button"
						class="w-full rounded-md px-3 py-2 text-left transition {selectedEnv === env.name
							? 'bg-[#24304a] text-white'
							: 'hover:bg-[#1d2130]'}"
						disabled={$isBusy}
						onclick={() => selectEnvironment(env.name)}
					>
						<div class="font-medium">{env.name}</div>
						<div class="muted text-xs">{env.provider} · {env.project}</div>
					</button>
				</li>
			{/each}
		</ul>
	</aside>

	<section class="card p-4">
		<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
			<div>
				<h2 class="text-lg font-semibold">
					{selectedEnv ? `Environment: ${selectedEnv}` : "Select an environment"}
				</h2>
			</div>
			<div class="flex flex-wrap gap-2">
				<input
					type="search"
					placeholder="Filter secrets…"
					bind:value={filter}
					class="min-w-[180px]"
				/>
				<button type="button" class="ghost" onclick={toggleMask}>
					{masked ? "Unmask" : "Mask"}
				</button>
				<button type="button" class="primary" onclick={openCreate} disabled={!selectedEnv || $isBusy}>
					New secret
				</button>
			</div>
		</div>

		{#if !selectedEnv}
			<p class="muted">No environment selected.</p>
		{:else if !secretsLoaded}
			<p class="muted">Loading secrets…</p>
		{:else if filteredSecrets.length === 0}
			<p class="muted">No secrets found.</p>
		{:else}
			<div class="overflow-x-auto">
				<table>
					<thead>
						<tr>
							<th>Env var</th>
							<th>Value</th>
							<th>Provider</th>
							<th>Ref</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						{#each filteredSecrets as row}
							<tr>
								<td>{row.envVar}</td>
								<td class="font-mono text-sm">{masked ? row.maskedValue : row.value}</td>
								<td>{row.provider}</td>
								<td class="text-sm">
									{row.ref ? `${row.refKind}:${row.ref}` : row.refKind}
								</td>
								<td class="text-right">
									<div class="flex justify-end gap-2">
										<button type="button" class="ghost" onclick={() => openView(row)}>View</button>
										<button
											type="button"
											class="ghost"
											disabled={row.refKind !== "secret-key"}
											onclick={() => openEdit(row)}>Edit</button
										>
										<button
											type="button"
											class="ghost"
											disabled={row.refKind !== "secret-key"}
											onclick={() => openDelete(row)}>Delete</button
										>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>

{#if $viewOpen}
	<div class="dialog-layer">
		<div use:melt={$viewOverlay} class="dialog-backdrop"></div>
		<div use:melt={$viewContent} class="dialog" role="dialog" aria-modal="true">
			<h3 class="text-lg font-semibold">View secret</h3>
			<p class="muted text-sm">{viewEnvVar}</p>
			<pre class="overflow-x-auto rounded-md bg-[#10131b] p-3 text-sm">{viewValue}</pre>
			<div class="actions">
				<button type="button" class="ghost" onclick={copyViewValue}>
					{viewCopied ? "Copied!" : "Copy"}
				</button>
				<button type="button" use:melt={$viewClose} class="ghost">Close</button>
			</div>
		</div>
	</div>
{/if}

{#if $editOpen}
	<div class="dialog-layer">
		<div use:melt={$editOverlay} class="dialog-backdrop"></div>
		<div use:melt={$editContent} class="dialog" role="dialog" aria-modal="true">
			<h3 class="text-lg font-semibold">Edit secret</h3>
			<div class="field">
				<label for="edit-value">Secret value</label>
				<textarea id="edit-value" rows="4" bind:value={editValue}></textarea>
			</div>
			<div class="actions">
				<button type="button" use:melt={$editClose} class="ghost" disabled={$isBusy}>Cancel</button>
				<button type="button" class="primary" onclick={submitEdit} disabled={$isBusy}>Save</button>
			</div>
		</div>
	</div>
{/if}

{#if $createOpen}
	<div class="dialog-layer">
		<div use:melt={$createOverlay} class="dialog-backdrop"></div>
		<div use:melt={$createContent} class="dialog" role="dialog" aria-modal="true">
			<h3 class="text-lg font-semibold">Create secret & mapping</h3>
			<div class="field">
				<label for="create-env-var">Env var</label>
				<input id="create-env-var" bind:value={createEnvVar} placeholder="ENV_VAR_NAME" />
			</div>
			<div class="field">
				<label for="create-secret-key">Secret key/id</label>
				<input id="create-secret-key" bind:value={createSecretKey} placeholder="provider secret key" />
			</div>
			<div class="field">
				<label for="create-value">Value</label>
				<textarea id="create-value" rows="3" bind:value={createValue}></textarea>
			</div>
			<div class="field">
				<label for="create-desc">Description (optional)</label>
				<input id="create-desc" bind:value={createDesc} />
			</div>
			{#if createOptions?.supportsReplication}
				<div class="field">
					<label for="create-replication">Replication</label>
					<select id="create-replication" bind:value={createReplication}>
						<option value="global">Global (automatic replication)</option>
						<option value="user-managed">User-managed (choose locations)</option>
					</select>
				</div>
				{#if createReplication === "user-managed"}
					<div class="field">
						<span>Locations</span>
						<div class="max-h-40 overflow-y-auto rounded-md border border-[#2a2f3d] p-2">
							{#each createOptions?.locations ?? [] as loc}
								<label class="flex items-center gap-2 py-1 text-sm">
									<input
										type="checkbox"
										checked={createLocations.includes(loc)}
										onchange={() => toggleLocation(loc)}
									/>
									{loc}
								</label>
							{/each}
						</div>
					</div>
				{/if}
			{/if}
			<div class="actions">
				<button type="button" use:melt={$createClose} class="ghost" disabled={$isBusy}>Cancel</button>
				<button type="button" class="primary" onclick={submitCreate} disabled={$isBusy}>Create</button>
			</div>
		</div>
	</div>
{/if}

{#if $deleteOpen}
	<div class="dialog-layer">
		<div use:melt={$deleteOverlay} class="dialog-backdrop"></div>
		<div use:melt={$deleteContent} class="dialog" role="dialog" aria-modal="true">
			<h3 class="text-lg font-semibold">Delete secret</h3>
			<p>
				Delete provider secret <strong>{deleteRef}</strong> and remove mapping for
				<strong>{deleteEnvVar}</strong>?
			</p>
			<div class="actions">
				<button type="button" use:melt={$deleteClose} class="ghost" disabled={$isBusy}>Cancel</button>
				<button type="button" class="danger" onclick={submitDelete} disabled={$isBusy}>Delete</button>
			</div>
		</div>
	</div>
{/if}

{#if $errorOpen}
	<div class="dialog-layer">
		<div use:melt={$errorOverlay} class="dialog-backdrop"></div>
		<div use:melt={$errorContent} class="dialog" role="alertdialog" aria-modal="true">
			<h3 class="text-lg font-semibold">Error</h3>
			<p class="error">{errorMessage}</p>
			<div class="actions">
				<button type="button" use:melt={$errorClose} class="primary">OK</button>
			</div>
		</div>
	</div>
{/if}
