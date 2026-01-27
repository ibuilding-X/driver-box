import {defineConfig} from 'astro/config';
import starlight from '@astrojs/starlight';
import mermaid from 'astro-mermaid';


// https://astro.build/config
export default defineConfig({
    site: 'https://ibuilding-X.github.io/driver-box/',
    base: '/driver-box',
    trailingSlash: "always",
    integrations: [mermaid({
        theme: 'forest',
        autoTheme: true
    }),
        starlight({
        title: 'driver-box',
        description: '一款嵌入式边缘平台',
        social:[
            { icon: 'github', label: 'GitHub', href: 'https://github.com/ibuilding-X/driver-box' }
        ],
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
            label: '应用组件(Export)',
            autogenerate: {
                directory: 'exports'
            }
        }, {
            label: '通信插件(Plugin)',
            autogenerate: {
                directory: 'plugins'
            }
        }, {
            label: '资源库(Library)',
            autogenerate: {
                directory: 'library'
            }
        }]
    })
    ]
});