import {defineConfig} from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
    site: 'https://roneli.github.io',
    integrations: [
        starlight({
            title: '',
            social: {
                github: 'https://github.com/roneli/fastgql',
            },
            logo: {
                dark: './src/assets/logo_light.svg',
                light: './src/assets/logo_dark.svg',
            },
            sidebar: [
                {
                    label: 'Getting Started',
                    items: [
                        {label: 'Introduction', link: '/start/intro'},
                        {label: 'Setup', link: '/start/setup'},
                    ]
                },
                {
                    label: 'Queries',
                    items: [
                        {label: 'Querying', link: '/queries/queries'},
                        {label: 'Filtering', link: '/queries/filtering'},
                        {label: 'Ordering', link: '/queries/ordering'},
                        {label: 'Pagination', link: '/queries/pagination'},
                        {label: 'Aggregation', link: '/queries/aggregation'},
                        ]
                },
                {
                    label: 'Mutations',
                    items: [
                        {label: 'Insert', link: '/mutations/insert'},
                        {label: 'Update', link: '/mutations/update'},
                        {label: 'Delete', link: '/mutations/delete'},
                    ]
                },
                {
                    label: 'Schema',
                    items: [
                        {label: 'Directives', link: '/schema/directives'},
                        {label: 'Schema', link: '/schema/fastgql_schema_fragment'},
                    ],
                },
                {
                    label: 'Reference',
                    autogenerate: {directory: 'reference'},
                },
            ],
        }),
    ],
});