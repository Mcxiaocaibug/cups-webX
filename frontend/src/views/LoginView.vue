<template>
  <div class="page-shell flex min-h-full items-center justify-center py-4 sm:py-8">
    <div class="grid w-full max-w-6xl gap-4 lg:grid-cols-[1.1fr_0.9fr]">
      <section class="hero-surface flex flex-col justify-between">
        <div class="space-y-5">
          <div class="section-kicker">Remote Print Hub</div>
          <div class="space-y-3">
            <h2 class="max-w-xl text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
              让打印机像内部系统一样稳定、直接、可追踪。
            </h2>
            <p class="max-w-2xl text-sm leading-6 text-slate-700 sm:text-base">
              浏览器上传文档后即可远程打印，自动保留任务记录、设备状态与管理入口。前台操作更顺手，后台排障也更快。
            </p>
          </div>
        </div>

        <div class="dashboard-grid mt-8">
          <div class="metric-card">
            <div class="metric-label">打印入口</div>
            <div class="metric-value">统一网页端</div>
            <div class="metric-helper">支持 PDF、图片、Office 与文本文件。</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">记录追踪</div>
            <div class="metric-value">任务可回溯</div>
            <div class="metric-helper">打印结果、页数、设备与状态都能留痕。</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">安全控制</div>
            <div class="metric-value">会话 + CSRF</div>
            <div class="metric-helper">登录后即可进入打印台，默认建议立即改密码。</div>
          </div>
        </div>
      </section>

      <UCard class="soft-card mx-auto w-full max-w-xl">
        <template #header>
          <div class="space-y-2">
            <div class="flex items-center gap-2 text-slate-900">
              <UIcon name="i-lucide-key-round" class="h-5 w-5 text-primary" />
              <h3 class="text-xl font-semibold">登录</h3>
            </div>
            <p class="text-sm app-muted">输入账号后进入打印控制台。</p>
          </div>
        </template>

        <UAlert
          v-if="error"
          icon="i-lucide-triangle-alert"
          color="error"
          variant="soft"
          class="mb-6"
        >
          {{ error }}
        </UAlert>

        <div class="soft-inline-note mb-6 p-4 text-sm text-slate-700">
          <p class="font-medium text-slate-900">部署提示</p>
          <p class="mt-1">如果是首次启动，系统会先自动跳转到部署向导，不需要提前手工配置管理员账号。</p>
        </div>

        <UForm @submit="login" :state="state" class="space-y-6">
          <UFormField label="用户名" name="username" required>
            <UInput v-model="state.username" icon="i-lucide-user" size="xl" class="w-full" placeholder="请输入用户名" />
          </UFormField>

          <UFormField label="密码" name="password" required>
            <UInput
              v-model="state.password"
              type="password"
              icon="i-lucide-lock"
              size="xl"
              class="w-full"
              placeholder="请输入密码"
            />
          </UFormField>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p class="text-xs app-muted">会话将写入安全 Cookie，浏览器刷新后自动恢复。</p>
            <UButton
              type="submit"
              color="primary"
              icon="i-lucide-log-in"
              size="lg"
              class="w-full sm:w-auto sm:min-w-36"
              :loading="loading"
            >
              进入系统
            </UButton>
          </div>
        </UForm>
      </UCard>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'

const state = reactive({
  username: '',
  password: ''
})
const error = ref('')
const loading = ref(false)

const emit = defineEmits(['login-success'])

async function login() {
  error.value = ''
  loading.value = true
  try {
    const resp = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: state.username, password: state.password }),
      credentials: 'include'
    })
    if (!resp.ok) {
      const data = await resp.json().catch(() => ({}))
      error.value = data.error || '登录失败'
      return
    }
    emit('login-success')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
