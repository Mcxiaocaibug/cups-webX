<template>
  <UApp>
    <div class="min-h-screen">
      <div class="app-shell grid min-h-screen grid-rows-[auto_1fr_auto] gap-4">
        <header class="soft-nav">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div class="flex items-start gap-4">
              <div class="status-pill bg-white/90">
                <UIcon name="i-lucide-printer" class="h-4 w-4 text-primary" />
                <span class="text-xs font-semibold tracking-[0.24em] text-slate-500">CUPS WEB</span>
              </div>
              <div>
                <h1 class="text-xl font-semibold text-slate-900">{{ setupStatus.setupComplete ? '浏览器打印控制台' : '首次部署向导' }}</h1>
                <p class="mt-1 text-sm app-muted">
                  {{ setupStatus.setupComplete ? '上传文档、查看打印机状态、维护打印记录。' : '容器启动后直接在网页端完成管理员和 CUPS 连接配置。' }}
                </p>
              </div>
            </div>

            <div class="flex flex-wrap items-center justify-end gap-2">
              <UBadge v-if="!setupLoading && !setupStatus.setupComplete" color="primary" variant="soft" size="lg">
                待初始化
              </UBadge>
              <UBadge v-if="session" color="neutral" variant="soft" size="lg">
                {{ session.username }} · {{ isAdmin ? '管理员' : '普通用户' }}
              </UBadge>
              <UBadge v-if="session?.mustChangePassword" color="warning" variant="soft" size="lg">
                需修改密码
              </UBadge>
              <UButton v-if="setupStatus.setupComplete && session" :variant="$route.path === '/print' ? 'solid' : 'ghost'" size="sm" @click="$router.push('/print')">
                打印
              </UButton>
              <UButton v-if="setupStatus.setupComplete && isAdmin" :variant="$route.path === '/admin' ? 'solid' : 'ghost'" size="sm" @click="$router.push('/admin')">
                管理
              </UButton>
              <UButton v-if="setupStatus.setupComplete && session" variant="outline" size="sm" icon="i-lucide-log-out" @click="logout">
                登出
              </UButton>
            </div>
          </div>
        </header>

        <main class="min-h-0">
          <router-view
            :session="session"
            :setup="setupStatus"
            @login-success="onLogin"
            @logout="onLogout"
            @setup-complete="onSetupComplete"
            @session-refresh="loadSession"
          />
        </main>

        <footer class="soft-footer flex flex-col gap-2 text-sm app-muted md:flex-row md:items-center md:justify-between">
          <p>{{ setupStatus.setupComplete ? '单体服务模式，前端构建后直接嵌入 Go 二进制。' : '首次部署无需预写环境变量，网页端会引导完成初始化。' }}</p>
          <p>
            Powered by
            <a href="https://github.com/hanxi/cups-web" target="_blank" class="font-medium text-slate-700 hover:text-slate-900">
              cups-web
            </a>
          </p>
        </footer>
      </div>
    </div>
  </UApp>
</template>

<script>
import LoginView from './views/LoginView.vue'
import PrintView from './views/PrintView.vue'
import AdminView from './views/AdminView.vue'
import SetupView from './views/SetupView.vue'

export default {
  data() {
    return {
      session: null,
      setupLoading: true,
      setupStatus: {
        setupComplete: true,
        cupsHost: '',
        adminCount: 0,
        adminUsername: 'admin'
      }
    }
  },
  async mounted() {
    await this.loadSetupState()
  },
  components: { LoginView, PrintView, AdminView, SetupView },
  computed: {
    isAdmin() {
      return this.session && this.session.role === 'admin'
    }
  },
  methods: {
    async loadSetupState() {
      this.setupLoading = true
      try {
        const resp = await fetch('/api/setup/status', { credentials: 'include' })
        if (resp.ok) {
          this.setupStatus = await resp.json()
        }
      } catch (e) {
        // keep fallback state
      } finally {
        this.setupLoading = false
      }

      if (!this.setupStatus.setupComplete) {
        this.session = null
        if (this.$route.path !== '/setup') {
          this.$router.push('/setup')
        }
        return
      }

      await this.loadSession()
    },
    async loadSession() {
      if (!this.setupStatus.setupComplete) {
        this.session = null
        return
      }
      try {
        const resp = await fetch('/api/session', { credentials: 'include' })
        if (resp.ok) {
          this.session = await resp.json()
          if (this.$route.path === '/' || this.$route.path === '/login' || this.$route.path === '/setup') {
            this.$router.push('/print')
          }
        } else {
          this.session = null
          if (this.$route.path !== '/login') {
            this.$router.push('/login')
          }
        }
      } catch (e) {
        this.session = null
        if (this.$route.path !== '/login') {
          this.$router.push('/login')
        }
      }
    },
    async onLogin() {
      await this.loadSession()
    },
    async onSetupComplete(payload) {
      this.setupStatus = {
        ...this.setupStatus,
        setupComplete: true,
        cupsHost: payload?.cupsHost || this.setupStatus.cupsHost,
        adminCount: 1
      }
      this.session = payload?.session || null
      if (!this.session) {
        await this.loadSession()
        return
      }
      this.$router.push('/print')
    },
    onLogout() {
      this.session = null
      this.$router.push('/login')
    },
    async logout() {
      try {
        await fetch('/api/logout', { method: 'POST', credentials: 'include' })
      } catch (e) {
        // ignore errors
      }
      this.onLogout()
    }
  }
}
</script>
