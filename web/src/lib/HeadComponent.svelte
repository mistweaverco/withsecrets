<script>
	export let data;
</script>

<svelte:head>
	<title>{data.title}</title>
	<meta name="description" content={data.description} />
	<link rel="icon" type="image/png" href="/favicon-96x96.png" sizes="96x96" />
	<link rel="icon" type="image/svg+xml" href="/favicon.svg" />
	<link rel="shortcut icon" href="/favicon.ico" />
	<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
	<link rel="manifest" href="/site.webmanifest" />
	<meta property="og:image" content="/open-graph.png" />
	<meta property="og:title" content={data.title} />
</svelte:head>
