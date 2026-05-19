import { pluginSass } from '@rsbuild/plugin-sass'
import { defineConfig } from '@rspress/core'
import { pluginLlms } from '@rspress/plugin-llms'
import { pluginScalar } from '@seshuk/rspress-plugin-scalar'
import * as path from 'node:path'

export default defineConfig({
  root: path.join(__dirname, 'docs'),
  base: process.env.DOCS_BASE ?? '/',
  title: 'snapr',
  description: 'Self-hosted backup service with web UI and REST API',
  icon: '/favicon.png',
  logo: {
    light: '/logo-dark.svg',
    dark: '/logo-light.svg',
  },
  logoText: 'snapr',
  globalStyles: path.join(__dirname, 'styles/custom.css'),
  plugins: [
    pluginLlms(),
    pluginScalar({
      instances: [
        {
          route: '/api',
          configuration: {
            url: '/openapi.json',
            mcp: { disabled: true },
            agent: { disabled: true },
          },
        },
      ],
    }),
  ],
  builderConfig: {
    plugins: [pluginSass()],
  },
  themeConfig: {
    editLink: {
      docRepoBaseUrl: 'https://github.com/maximseshuk/snapr/tree/main/docs/docs',
    },
    lastUpdated: true,
    llmsUI: {
      placement: 'outline',
      viewOptions: ['markdownLink', 'claude', 'chatgpt'],
    },
    socialLinks: [
      {
        icon: 'github',
        mode: 'link',
        content: 'https://github.com/maximseshuk/snapr',
      },
    ],
  },
})
