import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import SetupView from './views/SetupView.vue'
import LoginView from './views/LoginView.vue'
import PrintView from './views/PrintView.vue'
import AdminView from './views/AdminView.vue'
import './index.css'
import ui from '@nuxt/ui/vue-plugin'

const app = createApp(App)

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/setup', component: SetupView },
    { path: '/login', component: LoginView },
    { path: '/print', component: PrintView },
    { path: '/admin', component: AdminView }
  ]
})

app.use(router)
app.use(ui)
app.mount('#app')
