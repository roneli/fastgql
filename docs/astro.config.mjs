import {defineConfig} from 'astro/config';
import starlight from '@astrojs/starlight';
import vercel from '@astrojs/vercel/static';
import icon from "astro-icon";

import tailwind from "@astrojs/tailwind";

// https://astro.build/config
export default defineConfig({
    output: 'static',
    adapter: vercel({
        webAnalytics: {
            enabled: true
        },
        maxDuration: 8
    }),
    site: 'https://fastgql.com',
    base: "/",
    integrations: [starlight({
        title: 'FastGQL',
        favicon: './favicon.svg',
        social: {
            github: 'https://github.com/roneli/fastgql'
        },
        components: {
            // override ThemeProvider
            ThemeSelect: './src/components/ThemeSelect.astro',
        },
        logo: {
            dark: './src/assets/logo_dark.svg',
            light: './src/assets/logo_light.svg',
            replacesTitle: true
        },
        editLink: {
            baseUrl: 'https://github.com/roneli/fastgql/tree/master/docs',
        },
        customCss: ['./src/styles/global.css', './src/styles/landing.css'],
        lastUpdated: true,
        sidebar: [{
            label: 'Getting Started',
            items: [{
                label: 'Introduction',
                link: '/start/intro'
            }, {
                label: 'Setup',
                link: '/start/setup',
            }]
        }, {
            label: 'Queries',
            items: [{
                label: 'Querying',
                link: '/queries/queries'
            }, {
                label: 'Filtering',
                link: '/queries/filtering'
            }, {
                label: 'Ordering',
                link: '/queries/ordering'
            }, {
                label: 'Pagination',
                link: '/queries/pagination'
            }, {
                label: 'Aggregation',
                link: '/queries/aggregation'
            }]
        }, {
            label: 'Mutations',
            items: [{
                label: 'Insert',
                link: '/mutations/insert'
            }, {
                label: 'Update',
                link: '/mutations/update'
            }, {
                label: 'Delete',
                link: '/mutations/delete'
            }]
        }, {
            label: 'Schema',
            items: [
                {
                    label: 'Interfaces',
                    link: '/schema/interfaces',
                    badge: {text: 'Experimental', variant: 'caution'},
                },
                {
                    label: 'Custom Operators',
                    link: '/schema/operators',
                    badge: 'New',
                },
                {
                    label: 'Directives',
                    link: '/schema/directives'
                },
                {
                    label: 'Schema',
                    link: '/schema/schema'
                }]
        }, {
            label: 'Reference',
            autogenerate: {
                directory: 'reference'
            }
        }]
    }), icon(), tailwind()]
});