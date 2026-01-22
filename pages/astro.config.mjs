import {defineConfig} from 'astro/config';
import starlight from '@astrojs/starlight';
import remarkMermaid from 'remark-mermaidjs'
import mdx from '@astrojs/mdx';
import expressiveCode from "astro-expressive-code";


// https://astro.build/config
export default defineConfig({
    site: 'https://ibuilding-X.github.io/',
    base: '/driver-box',
    trailingSlash: "always",
    markdown: {
        // Applied to .md and .mdx files
        remarkPlugins: [remarkMermaid],
    },
    integrations: [expressiveCode(), mdx(),starlight({
        title: 'driver-box',
        social: {
            github: 'https://github.com/ibuilding-X/driver-box'
        },
        head: [{
            tag: 'script',
            content: `
                var _hmt = _hmt || [];
                (function() {
                  var hm = document.createElement("script");
                  hm.src = "https://hm.baidu.com/hm.js?81f653be99c4697c95cedbdacc3023b4";
                  var s = document.getElementsByTagName("script")[0]; 
                  s.parentNode.insertBefore(hm, s);
                })();
          `
        }],
        sidebar: [{
            label: '使用指南',
            autogenerate: {
                directory: 'guides'
            }
            // items: [
            // 	// Each item here is one entry in the navigation menu.
            // 	{ label: '项目简介', link: '/guides/example/' },
            // ],
        }, {
            label: '核心概念',
            autogenerate: {
                directory: 'concepts'
            }
        }, {
            label: '插件',
            autogenerate: {
                directory: 'plugins'
            }
        }, {
            label: 'Export',
            autogenerate: {
                directory: 'export'
            }
        }, {
            label: '资产库',
            autogenerate: {
                directory: 'library'
            }
        }, {
            label: '开发指南',
            autogenerate: {
                directory: 'developer'
            }
        }]
    })]
});