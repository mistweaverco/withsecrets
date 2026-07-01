export type SearchEntry = {
	title: string;
	href: string;
	/** Extra terms users might type */
	keywords: string[];
	/** Short hint shown in results */
	excerpt?: string;
};

export const SEARCH_INDEX: SearchEntry[] = [
	{
		title: "Home",
		href: "/",
		keywords: ["home"],
		excerpt: "The home page.",
	},
	{
		title: "Installation",
		href: "/installation",
		keywords: [
			"arch",
			"pkgbuild",
			"withsecrets-bin",
			"pacman",
			"makepkg",
			"aur",
			"paru",
			"yay",
			"install",
			"download",
			"linux",
			"macos",
			"windows",
			"powershell",
			"curl",
			"wget",
			"zsh",
			"bash",
		],
		excerpt:
			"Install withsecrets CLI on Arch (withsecrets-bin), other Linux distros, macOS, and Windows.",
	},
	{
		title: "Usage",
		href: "/usage",
		keywords: ["run", "contain", "test", "show", "update", "debug", "docker", "ci", "cd"],
	},
	{
		title: "Configuration",
		href: "/configuration",
		keywords: [
			"ws.yaml",
			"schema",
			"env",
			"secret-key",
			"secret-path",
			"value",
			"interpolation",
			"convert",
			"init",
		],
		excerpt:
			"How to configure withsecrets with ws.yaml, including schema, env interpolation, and secret management.",
	},
	{
		title: "Getting Started",
		href: "/configuration#getting-started",
		keywords: [
			"convert",
			"ksvc",
			"dotenv",
			".env",
			"cloud run",
			"knative",
			"remote import",
			"provider",
			"gcp",
			"aws",
			"azure",
			"app runner",
			"container apps",
		],
		excerpt:
			"Getting started guide for converting existing .env files or remote secrets into a ws.yaml configuration.",
	},
	{
		title: "Import from dotenv (.env*)",
		href: "/configuration#import-from-dotenv",
		keywords: [
			"convert",
			"ksvc",
			"dotenv",
			".env",
			"cloud run",
			"knative",
			"remote import",
			"provider",
			"gcp",
			"aws",
			"azure",
			"app runner",
			"container apps",
		],
		excerpt:
			"Use ws convert dotenv to convert an existing .env file into a ws.yaml configuration, with support for remote secrets and providers.",
	},
	{
		title: "Import from Knative Services (ksvc)",
		href: "/configuration#import-from-knative-service-ksvc",
		keywords: [
			"convert",
			"ksvc",
			"dotenv",
			".env",
			"cloud run",
			"knative",
			"remote import",
			"provider",
			"gcp",
			"aws",
			"azure",
			"app runner",
			"container apps",
		],
		excerpt:
			"Use ws convert ksvc to convert an existing Knative Service (ksvc) into a ws.yaml configuration, with support for remote secrets and providers.",
	},
	{
		title: "Templating ws.yaml",
		href: "/configuration#withsecrets-global-templates",
		keywords: ["template", "configuration"],
		excerpt:
			"Use global templates to avoid repeating common configuration patterns in your ws.yaml, and to create reusable building blocks for your secrets management.",
	},
	{
		title: "withsecrets defaults",
		href: "/configuration#withsecrets-global-config-defaults",
		keywords: ["template", "configuration", "defaults", "region"],
		excerpt: "Setting defaults per provider that is used when you run ws tui",
	},
	{
		title: "Import from already deployed service",
		href: "/configuration#import-from-knative-service-ksvc-from-deployed-service",
		keywords: [
			"convert",
			"ksvc",
			"dotenv",
			".env",
			"cloud run",
			"knative",
			"remote import",
			"provider",
			"gcp",
			"aws",
			"azure",
			"app runner",
			"container apps",
		],
		excerpt:
			"Use ws convert --from ksvc to convert an already deployed Knative Service into a ws.yaml configuration.",
	},
	{
		title: "Providers",
		href: "/providers",
		keywords: [
			"providers",
			"gcp",
			"aws",
			"azure",
			"openbao",
			"bitwarden",
			"local",
			"auth",
			"permissions",
		],
		excerpt:
			"Configure providers to fetch secrets from external sources like GCP Secret Manager, AWS Secrets Manager, Azure Key Vault, OpenBAO, Bitwarden, or local files.",
	},
	{
		title: "Examples",
		href: "/examples",
		keywords: ["examples"],
		excerpt: "Find example ws.yaml configurations",
	},
	{
		title: "Cross provider examples",
		href: "/examples#cross-provider-configuration",
		keywords: ["examples", "cross provider"],
		excerpt:
			"Example ws.yaml showing how to use multiple providers together to fetch secrets from GCP, AWS, and Azure in the same configuration.",
	},
	{
		title: "Github Actions examples",
		href: "/examples#github-actions",
		keywords: ["examples", "github"],
		excerpt: "Example github-actions workflow using withsecrets to inject secrets into a CI job.",
	},
	{
		title: "GitLab CI examples",
		href: "/examples#gitlab-ci",
		keywords: ["examples", "gitlab"],
		excerpt: "Example .gitlab-ci.yml using withsecrets to inject secrets into a CI job.",
	},
	{
		title: "Node.js examples",
		href: "/examples#nodejs-express-application",
		keywords: ["examples", "nodejs", "express", "node", "typescript"],
		excerpt: "Example ws.yaml for running a Node.js Express application with secrets injected.",
	},
	{
		title: "Python examples",
		href: "/examples#python-flask-application",
		keywords: ["examples", "python"],
		excerpt: "Example ws.yaml for running a Python Flask application with secrets injected.",
	},
	{
		title: "Docker examples",
		href: "/examples#docker-and-container-examples",
		keywords: ["examples", "docker"],
		excerpt: "Example ws.yaml for running a Docker container with secrets injected.",
	},
	{
		title: "Docker Compose examples",
		href: "/examples#docker-compose-integration",
		keywords: ["examples", "docker", "compose"],
		excerpt: "Example ws.yaml for running a Docker container with secrets injected.",
	},
	{
		title: "Interactive TUI",
		href: "/usage#tui",
		keywords: ["tui", "interactive", "secrets", "edit", "add", "terminal ui"],
		excerpt: "Use ws tui to view, add, and edit secrets interactively.",
	},
	{
		title: "Changelog (CLI)",
		href: "/usage#changelog",
		keywords: ["changelog", "release notes", "latest", "version"],
		excerpt: "Use ws changelog to view the baked-in changelog in your terminal.",
	},
	{
		title: "Create Template",
		href: "/usage#create-template",
		keywords: ["create", "template", "editor", "VISUAL", "EDITOR", "ws create template"],
		excerpt: "Use ws create template to create or edit your user template in $EDITOR.",
	},
];
