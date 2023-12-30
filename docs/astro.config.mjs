import {defineConfig} from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
    site: 'https://roneli.github.io',
    base: "/fastgql",
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
                        {label: 'Introduction', link: '/fastgql/start/intro'},
                        {label: 'Setup', link: '/fastgql/start/setup'},
                    ]
                },
                {
                    label: 'Queries',
                    items: [
                        {label: 'Querying', link: '/fastgql/queries/queries'},
                        {label: 'Filtering', link: '/fastgql/queries/filtering'},
                        {label: 'Ordering', link: '/fastgql/queries/ordering'},
                        {label: 'Pagination', link: '/fastgql/queries/pagination'},
                        {label: 'Aggregation', link: '/fastgql/queries/aggregation'},
                        ]
                },
                {
                    label: 'Mutations',
                    items: [
                        {label: 'Insert', link: '/fastgql/mutations/insert'},
                        {label: 'Update', link: '/fastgql/mutations/update'},
                        {label: 'Delete', link: '/fastgql/mutations/delete'},
                    ]
                },
                {
                    label: 'Schema',
                    items: [
                        {label: 'Directives', link: '/fastgql/schema/directives'},
                        {label: 'Schema', link: '/fastgql/schema/fastgql_schema_fragment'},
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
