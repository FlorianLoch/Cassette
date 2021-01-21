import Vue from 'vue'
import App from './App.vue'
import router from './router'
import API from './lib/API'
import axios from "axios"
import NProgress from "nprogress"
import { BootstrapVue } from 'bootstrap-vue'

import 'nprogress/nprogress.css'
import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-vue/dist/bootstrap-vue.css'
import './styles.scss'

Vue.use(API)
Vue.use(BootstrapVue)

NProgress.configure({ showSpinner: false });

axios.interceptors.request.use((config) => {
  NProgress.start()
  return config
})
axios.interceptors.response.use((response) => {
  NProgress.done()
  return response
})

new Vue({
  router,
  render: h => h(App)
}).$mount('#app')