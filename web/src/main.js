import Vue from "vue"
import App from "./App.vue"
import router from "./router"
import API from "./lib/API"
import axios from "axios"
import NProgress from "nprogress"
import {
    ModalPlugin,
    DropdownPlugin,
    BButton,
    ProgressPlugin,
    CollapsePlugin,
} from "bootstrap-vue"

import "./styles.scss"

const gitVersion = process.env.GIT_VERSION
const gitAuthorDate = process.env.GIT_AUTHOR_DATE
const buildDate = process.env.BUILD_DATE

console.info(
    `UI based on git commit: ${gitVersion}, authored at ${gitAuthorDate}, built at ${buildDate}`
)

axios.interceptors.request.use((config) => {
    NProgress.start()
    return config
})
axios.interceptors.response.use(
    (response) => {
        NProgress.done()
        return response
    },
    (err) => {
        NProgress.done()
        return Promise.reject(err)
    }
)

Vue.use(API, { axios })
Vue.use(ModalPlugin)
Vue.use(DropdownPlugin)
Vue.use(ProgressPlugin)
Vue.use(CollapsePlugin)
Vue.component("BButton", BButton)

NProgress.configure({ showSpinner: false })

if (!API.consentGiven()) {
    router.replace({ name: "Consent" })
}

new Vue({
    router,
    render: (h) => h(App),
}).$mount("#app")
