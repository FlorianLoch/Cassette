import Vue from 'vue'
import App from './App.vue'
import router from './router'
import API from './lib/API'
import axios from "axios"
import NProgress from "nprogress"
import { ModalPlugin, DropdownPlugin, BButton } from 'bootstrap-vue'

import './styles.scss'

axios.interceptors.request.use((config) => {
  NProgress.start()
  return config
})
axios.interceptors.response.use((response) => {
  NProgress.done()
  return response
}, (err) => {
  NProgress.done()
  return Promise.reject(err)
})

Vue.use(API, {axios})
Vue.use(ModalPlugin)
Vue.use(DropdownPlugin)
Vue.component('b-button', BButton)

NProgress.configure({ showSpinner: false });

if (!API.isConsentCookieValid()) {
  router.replace({name: "Consent"})
}

new Vue({
  router,
  render: h => h(App)
}).$mount('#app')