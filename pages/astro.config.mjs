import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'Driver-Box',
			social: {
				github: 'https://github.com/withastro/starlight',
			},
			sidebar: [
				{
					label: '快速上手',
          autogenerate: { directory: 'guides' },
					// items: [
					// 	// Each item here is one entry in the navigation menu.
					// 	{ label: '项目简介', link: '/guides/example/' },
					// ],
				},
				{
					label: '插件',
					autogenerate: { directory: 'plugins' },
				},
			],
		}),
	],
});