import adapter from "@sveltejs/adapter-static";
import { preprocessMeltUI, sequence } from "@melt-ui/pp";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: sequence([vitePreprocess(), preprocessMeltUI()]),
	kit: {
		adapter: adapter({
			fallback: "index.html"
		}),
		alias: {
			$lib: "src/lib"
		}
	}
};

export default config;
