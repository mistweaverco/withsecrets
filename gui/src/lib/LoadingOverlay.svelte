<script lang="ts">
	import { busyMessage } from "$lib/busy";
</script>

{#if $busyMessage}
	<div class="loading-overlay" role="status" aria-live="polite" aria-busy="true">
		<div class="loading-panel card">
			<div class="loading-spinner" aria-hidden="true"></div>
			<p class="loading-text">{$busyMessage}</p>
		</div>
	</div>
{/if}
