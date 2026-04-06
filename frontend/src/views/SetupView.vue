<template>
  <div class="page-shell flex min-h-full items-center justify-center py-4 sm:py-8">
    <div class="grid w-full max-w-6xl gap-4 lg:grid-cols-[1.05fr_0.95fr]">
      <section class="hero-surface flex flex-col justify-between">
        <div class="space-y-5">
          <div class="section-kicker">Guided Setup</div>
          <div class="space-y-3">
            <h2 class="max-w-xl text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
              拉起容器后，直接在网页端完成初始化。
            </h2>
            <p class="max-w-2xl text-sm leading-6 text-slate-700 sm:text-base">
              不再要求先手工准备默认管理员、CUPS 地址或额外配置文件。先连上 CUPS，再设定管理员密码，初始化结束后会自动进入工作台。
            </p>
          </div>
        </div>

        <div class="dashboard-grid mt-8">
          <div class="metric-card">
            <div class="metric-label">步骤 1</div>
            <div class="metric-value">连接 CUPS</div>
            <div class="metric-helper">支持 `cups:631`、`host.docker.internal:631` 或局域网 IP。</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">步骤 2</div>
            <div class="metric-value">设定管理员</div>
            <div class="metric-helper">初始化时直接写入安全密码，不再生成 `admin/admin`。</div>
          </div>
          <div class="metric-card">
            <div class="metric-label">步骤 3</div>
            <div class="metric-value">开始打印</div>
            <div class="metric-helper">完成后自动登录，无需再额外回到登录页。</div>
          </div>
        </div>
      </section>

      <UCard class="soft-card mx-auto w-full max-w-xl">
        <template #header>
          <div class="space-y-2">
            <div class="flex items-center gap-2 text-slate-900">
              <UIcon name="i-lucide-wrench" class="h-5 w-5 text-primary" />
              <h3 class="text-xl font-semibold">部署向导</h3>
            </div>
            <p class="text-sm app-muted">当前实例尚未初始化。完成下面配置后会自动创建管理员并进入系统。</p>
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
          <p class="font-medium text-slate-900">地址示例</p>
          <p class="mt-1">同一 Docker 网络可填 <code>cups:631</code>；宿主机上的 CUPS 可填 <code>host.docker.internal:631</code> 或局域网地址。</p>
        </div>

        <UForm @submit="completeSetup" :state="state" class="space-y-6">
          <UFormField label="CUPS 地址" name="cupsHost" required>
            <UInput v-model="state.cupsHost" icon="i-lucide-printer" size="xl" class="w-full" placeholder="例如 cups:631" />
          </UFormField>

          <UFormField label="管理员密码" name="adminPassword" required>
            <UInput
              v-model="state.adminPassword"
              type="password"
              icon="i-lucide-lock"
              size="xl"
              class="w-full"
              placeholder="至少 6 位"
            />
          </UFormField>

          <UFormField label="确认管理员密码" name="confirmPassword" required>
            <UInput
              v-model="state.confirmPassword"
              type="password"
              icon="i-lucide-shield-check"
              size="xl"
              class="w-full"
              placeholder="再次输入密码"
            />
          </UFormField>

          <UFormField label="记录保留天数" name="retentionDays" hint="0 表示不自动清理，建议先保留 30 天">
            <UInput v-model.number="state.retentionDays" type="number" min="0" size="xl" class="w-full" placeholder="例如 30" />
          </UFormField>

          <div v-if="cupsCheck.message" class="soft-inline-note p-4 text-sm" :class="cupsCheck.ok ? 'text-success' : 'text-error'">
            {{ cupsCheck.message }}
          </div>

          <div v-if="cupsCheck.ok && cupsCheck.printers.length" class="rounded-3xl border border-slate-200/70 bg-white/75 p-4">
            <p class="text-sm font-medium text-slate-900">检测到的打印机</p>
            <div class="mt-3 flex flex-wrap gap-2">
              <div v-for="printer in cupsCheck.printers.slice(0, 8)" :key="printer.uri" class="status-pill text-xs app-muted">
                {{ printer.name }}
              </div>
            </div>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p class="text-xs app-muted">初始化完成后会自动保存 CUPS 地址并创建管理员会话。</p>
            <div class="flex w-full gap-2 sm:w-auto">
              <UButton
                type="button"
                variant="soft"
                icon="i-lucide-plug-zap"
                size="lg"
                class="flex-1 sm:flex-none"
                :loading="testing"
                @click="testConnection"
              >
                测试连接
              </UButton>
              <UButton
                type="submit"
                color="primary"
                icon="i-lucide-check-check"
                size="lg"
                class="flex-1 sm:min-w-36"
                :loading="submitting"
              >
                完成部署
              </UButton>
            </div>
          </div>
        </UForm>
      </UCard>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'

const props = defineProps({
  setup: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['setup-complete'])
const toast = useToast()

const state = reactive({
  cupsHost: '',
  adminPassword: '',
  confirmPassword: '',
  retentionDays: 30
})

const testing = ref(false)
const submitting = ref(false)
const error = ref('')
const cupsCheck = ref({
  ok: false,
  message: '',
  printers: []
})

watch(() => props.setup?.cupsHost, value => {
  if (!state.cupsHost && value) {
    state.cupsHost = value
  }
}, { immediate: true })

async function testConnection() {
  error.value = ''
  testing.value = true
  cupsCheck.value = { ok: false, message: '', printers: [] }
  try {
    const resp = await fetch('/api/setup/test-cups', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cupsHost: state.cupsHost.trim() })
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok || data.ok === false) {
      throw new Error(data.error || resp.statusText)
    }
    cupsCheck.value = {
      ok: true,
      message: `连接正常，发现 ${data.printerCount || 0} 台打印机。`,
      printers: data.printers || []
    }
    toast.add({ title: 'CUPS 连接正常', color: 'success' })
  } catch (e) {
    cupsCheck.value = {
      ok: false,
      message: `连接失败：${e.message}`,
      printers: []
    }
    error.value = e.message
  } finally {
    testing.value = false
  }
}

async function completeSetup() {
  error.value = ''
  if (!state.cupsHost.trim()) {
    error.value = '请输入 CUPS 地址'
    return
  }
  if (state.adminPassword.length < 6) {
    error.value = '管理员密码至少需要 6 位字符'
    return
  }
  if (state.adminPassword !== state.confirmPassword) {
    error.value = '两次输入的管理员密码不一致'
    return
  }

  submitting.value = true
  try {
    const resp = await fetch('/api/setup/bootstrap', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        cupsHost: state.cupsHost.trim(),
        adminPassword: state.adminPassword,
        confirmPassword: state.confirmPassword,
        retentionDays: Number.isFinite(state.retentionDays) ? Math.max(0, Number(state.retentionDays)) : 30
      })
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      throw new Error(data.error || resp.statusText)
    }
    emit('setup-complete', data)
  } catch (e) {
    error.value = e.message
  } finally {
    submitting.value = false
  }
}
</script>
