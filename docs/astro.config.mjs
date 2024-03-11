import {defineConfig} from 'astro/config';
import starlight from '@astrojs/starlight';
import vercel from '@astrojs/vercel/static';
import image from '@astrojs/image'
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
        title: '',
        social: {
            github: 'https://github.com/roneli/fastgql'
        },
        components: {
            // override ThemeProvider
            ThemeSelect: './src/components/ThemeSelect.astro',
        },
        logo: {
            dark: './src/assets/logo_light.svg',
            light: './src/assets/logo_dark.svg',
            replacesTitle: false
        },
        editLink: {
            baseUrl: 'https://github.com/roneli/fastgql/tree/master/docs',
        },
        lastUpdated: true,
        sidebar: [{
            label: 'Getting Started',
            items: [{
                label: 'Introduction',
                link: '/start/intro'
            }, {
                label: 'Setup',
                link: '/start/setup'
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
            items: [{
                label: 'Directives',
                link: '/schema/directives'
            }, {
                label: 'Schema',
                link: '/schema_schema_fragment'
            }]
        }, {
            label: 'Reference',
            autogenerate: {
                directory: 'reference'
            }
        }]
    }), icon(), tailwind()
        image({ serviceEntryPoint: '@astrojs/image/sharp' })
    ]
});