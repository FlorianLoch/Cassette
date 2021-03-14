import Vue from 'vue'
import VueRouter from 'vue-router'
import Main from '../views/Main.vue'
import Consent from '../views/Consent.vue'

Vue.use(VueRouter)

const routes = [
  {
    path: '/',
    name: 'Main',
    component: Main
  },
  {
    path: '/yourData',
    name: 'Consent',
    component: Consent
  }
]

const router = new VueRouter({
  mode: 'history',
  routes
})

export default router