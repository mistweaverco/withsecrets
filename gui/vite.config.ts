import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite-plus";
import { sveltekit } from "@sveltejs/kit/vite";

export default defineConfig({
	plugins: [sveltekit(), tailwindcss()],
	server: {
		proxy: {
			"/api": {
				target: "http://127.0.0.1:11911",
				changeOrigin: true
			}
		}
	}
});
