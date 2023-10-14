import { defineConfig } from 'vite'
// TODO: Add again when migrating to Vue3
// import vue from '@vitejs/plugin-vue'
import { createVuePlugin as vue } from "vite-plugin-vue2";
import { fileURLToPath, URL } from "node:url"
import viteHtmlResolveAlias from "vite-plugin-html-resolve-alias"

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [vue(), viteHtmlResolveAlias()],
    build: {
        target: "es2015",
        minify: "terser",
    },
    resolve: {
        alias: {
            "@": fileURLToPath(new URL("./src", import.meta.url)),
        },
    },
    server: {
        proxy: {
            "/api": {
                target: "http://localhost:8082",
            },
        },
    },
    define: {
        "process.env": {
            GIT_VERSION: process.env.GIT_VERSION,
            GIT_AUTHOR_DATE: process.env.GIT_AUTHOR_DATE,
            BUILD_DATE: process.env.BUILD_DATE
        },
    },
})