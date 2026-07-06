<script lang="ts">
	let { children } = $props();
</script>

<svelte:head>
	<title>withsecrets GUI</title>
</svelte:head>

<div class="min-h-screen">
	<header class="border-b border-[#2a2f3d] px-6 py-4">
		<h1 class="text-xl font-semibold">withsecrets</h1>
		<p class="muted text-sm">Manage environments and secrets</p>
	</header>
	<main class="p-6">{@render children()}</main>
</div>
