<template>
  <div class="page-shell space-y-4 p-1 sm:p-2">
    <section class="hero-surface">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div class="space-y-3">
          <div class="section-kicker">Admin Console</div>
          <div>
            <h2 class="text-2xl font-semibold text-slate-900 sm:text-3xl">管理后台</h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-700">
              统一维护用户、打印记录与清理策略。这里优先提供排障信息和操作效率，而不是堆字段。
            </p>
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <UButton variant="outline" icon="i-lucide-refresh-cw" :loading="loadingUsers || loadingRecords" @click="refreshAll">
            刷新数据
          </UButton>
          <UButton variant="soft" icon="i-lucide-download" @click="exportRecordsCSV">
            导出筛选结果
          </UButton>
        </div>
      </div>

      <div class="dashboard-grid mt-6">
        <div class="metric-card">
          <div class="metric-label">用户总数</div>
          <div class="metric-value">{{ users.length }}</div>
          <div class="metric-helper">{{ adminCount }} 个管理员，{{ userCount }} 个普通用户。</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">打印记录</div>
          <div class="metric-value">{{ printRecords.length }}</div>
          <div class="metric-helper">今日 {{ todayPrintCount }} 条，筛选后 {{ filteredPrintRecords.length }} 条。</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">待处理任务</div>
          <div class="metric-value">{{ activePrintCount }}</div>
          <div class="metric-helper">排队 {{ queuedPrintCount }} 条，处理中 {{ processingPrintCount }} 条，关注 {{ staleActivePrintCount }} 条。</div>
        </div>
      </div>
    </section>

    <UCard class="soft-card">
      <template #header>
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h3 class="flex items-center gap-2 text-xl font-semibold text-slate-900">
              <UIcon name="i-lucide-users" class="h-5 w-5 text-primary" />
              用户管理
            </h3>
            <p class="mt-1 text-sm app-muted">支持新增、编辑、删除账号，并保留管理员保护规则。</p>
          </div>
          <UInput v-model="userSearch" icon="i-lucide-search" placeholder="搜索用户名、联系人、邮箱" class="w-full lg:max-w-sm" />
        </div>
      </template>

      <UForm @submit="saveUser" :state="form" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
        <UFormField label="登录名" name="username" required>
          <UInput v-model="form.username" :disabled="form.protected" placeholder="登录名" class="w-full" />
        </UFormField>

        <UFormField :label="isEditing ? '重置密码' : '初始密码'" name="password" :hint="isEditing ? '留空表示保持原密码' : '建议至少 6 位'">
          <UInput type="password" v-model="form.password" :placeholder="isEditing ? '留空不修改密码' : '密码'" class="w-full" />
        </UFormField>

        <UFormField label="角色" name="role" required>
          <USelect
            v-model="form.role"
            :disabled="form.protected"
            :items="roleItems"
            value-key="value"
            label-key="label"
            class="w-full"
          />
        </UFormField>

        <UFormField label="联系人" name="contactName">
          <UInput v-model="form.contactName" placeholder="联系人" class="w-full" />
        </UFormField>

        <UFormField label="联系电话" name="phone">
          <UInput v-model="form.phone" placeholder="联系电话" class="w-full" />
        </UFormField>

        <UFormField label="邮箱" name="email">
          <UInput v-model="form.email" placeholder="邮箱" class="w-full" />
        </UFormField>

        <div class="md:col-span-2 xl:col-span-2 flex flex-wrap items-end gap-2">
          <UButton type="submit" color="primary" icon="i-lucide-save" :loading="savingUser">
            {{ isEditing ? '保存修改' : '新增用户' }}
          </UButton>
          <UButton type="button" variant="outline" icon="i-lucide-rotate-ccw" @click="resetForm">
            重置
          </UButton>
          <span v-if="form.protected" class="text-xs app-muted">默认 admin 账号不能改名或降级。</span>
        </div>
      </UForm>

      <div class="mt-5 overflow-x-auto">
        <UTable :columns="userColumns" :data="filteredUsers">
          <template #role-data="{ row }">
            <UBadge :color="row.role === 'admin' ? 'primary' : 'neutral'" variant="soft" size="xs">
              {{ roleText(row.role) }}
            </UBadge>
          </template>

          <template #mustChangePassword-data="{ row }">
            <UBadge :color="row.mustChangePassword ? 'warning' : 'success'" variant="soft" size="xs">
              {{ row.mustChangePassword ? '待改密' : '已生效' }}
            </UBadge>
          </template>

          <template #createdAt-data="{ row }">
            <span class="text-xs app-muted">{{ formatTime(row.createdAt) }}</span>
          </template>

          <template #actions-data="{ row }">
            <div class="flex gap-2">
              <UButton size="xs" variant="ghost" icon="i-lucide-pencil" @click="editUser(row)">编辑</UButton>
              <UButton
                size="xs"
                variant="outline"
                color="error"
                icon="i-lucide-trash-2"
                :disabled="row.username === 'admin'"
                @click="deleteUser(row)"
              >
                删除
              </UButton>
            </div>
          </template>
        </UTable>
      </div>
    </UCard>

    <UCard class="soft-card">
      <template #header>
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h3 class="flex items-center gap-2 text-xl font-semibold text-slate-900">
              <UIcon name="i-lucide-file-text" class="h-5 w-5 text-primary" />
              打印记录
            </h3>
            <p class="mt-1 text-sm app-muted">支持按用户、日期、状态和关键字筛选，便于导出与排障。</p>
          </div>
        </div>
      </template>

      <div class="soft-table-tools mb-4">
        <UInput v-model="printFilters.username" icon="i-lucide-user-search" placeholder="用户名" />
        <UInput v-model="recordSearch" icon="i-lucide-search" placeholder="搜索文件名、任务 ID、打印机" />
        <USelect v-model="recordStatus" :items="statusItems" value-key="value" label-key="label" />
        <UInput type="date" v-model="printFilters.start" />
        <UInput type="date" v-model="printFilters.end" />
        <UButton variant="outline" icon="i-lucide-filter" :loading="loadingRecords" @click="loadPrintRecords">应用日期筛选</UButton>
      </div>

      <div class="mb-4 flex flex-wrap gap-2">
        <div class="status-pill text-xs app-muted">筛选结果：{{ filteredPrintRecords.length }} 条</div>
        <div class="status-pill text-xs app-muted">失败：{{ filteredPrintRecords.filter(r => r.status === 'failed').length }} 条</div>
        <div class="status-pill text-xs app-muted">排队中：{{ filteredPrintRecords.filter(r => r.status === 'queued').length }} 条</div>
        <div class="status-pill text-xs app-muted">处理中：{{ filteredPrintRecords.filter(r => r.status === 'processing').length }} 条</div>
        <div class="status-pill text-xs app-muted">需关注：{{ filteredPrintRecords.filter(r => isRecordStale(r)).length }} 条</div>
      </div>

      <div class="soft-inline-note mb-4 p-4">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <div class="text-sm font-semibold text-slate-900">问题聚类</div>
            <p class="mt-1 text-xs app-muted">把失败原因和长时间无变化的任务压成可点击筛选，方便直接聚焦一类问题。</p>
          </div>
          <UButton
            v-if="activeIssueFilterLabel"
            variant="ghost"
            size="xs"
            icon="i-lucide-filter-x"
            @click="clearIssueFilter"
          >
            清除问题筛选
          </UButton>
        </div>

        <div class="mt-3 flex flex-wrap gap-2">
          <UButton
            v-for="bucket in issueBuckets.slice(0, 8)"
            :key="bucket.key"
            size="xs"
            :variant="recordIssueFilter === bucket.key ? 'solid' : 'soft'"
            :color="bucket.color"
            @click="toggleIssueFilter(bucket.key)"
          >
            {{ bucket.label }} {{ bucket.count }}
          </UButton>
          <div v-if="issueBuckets.length === 0" class="text-xs app-muted">
            当前筛选范围内没有需要重点排查的问题任务。
          </div>
        </div>

        <div v-if="activeIssueFilterLabel" class="mt-3 text-xs app-muted">
          当前问题筛选：{{ activeIssueFilterLabel }}
        </div>
      </div>

      <div class="overflow-x-auto">
        <UTable :columns="printColumns" :data="filteredPrintRecords">
          <template #createdAt-data="{ row }">
            <span class="text-xs app-muted">{{ formatTime(row.createdAt) }}</span>
          </template>

          <template #filename-data="{ row }">
            <div class="min-w-0">
              <div class="truncate text-sm font-medium text-slate-900">{{ row.filename }}</div>
              <div class="mt-1 truncate text-xs app-muted">{{ formatRecordSummary(row) }}</div>
              <div v-if="row.statusDetail" class="mt-1 line-clamp-2 text-xs" :class="statusDetailClass(row.status)">{{ row.statusDetail }}</div>
            </div>
          </template>

          <template #status-data="{ row }">
            <UBadge :color="statusColor(row.status)" variant="soft" size="xs">
              {{ statusText(row.status) }}
            </UBadge>
          </template>

          <template #actions-data="{ row }">
            <div class="flex flex-wrap gap-2">
              <UButton size="xs" variant="ghost" :href="`/api/print-records/${row.id}/file`" target="_blank" icon="i-lucide-download">
                下载
              </UButton>
              <UButton
                size="xs"
                variant="soft"
                icon="i-lucide-eye"
                @click="selectRecord(row)"
              >
                详情
              </UButton>
              <UButton
                size="xs"
                variant="ghost"
                icon="i-lucide-rotate-cw"
                :loading="isRecordActionLoading(row.id, 'retry')"
                @click="retryRecord(row)"
              >
                重打
              </UButton>
              <UButton
                v-if="['queued', 'processing'].includes(row.status) && row.jobId"
                size="xs"
                variant="outline"
                color="warning"
                icon="i-lucide-ban"
                :loading="isRecordActionLoading(row.id, 'cancel')"
                @click="cancelRecord(row)"
              >
                取消
              </UButton>
            </div>
          </template>
        </UTable>
      </div>

      <div v-if="selectedRecord" class="mt-4 border-t border-slate-200/70 pt-4">
        <div class="mb-3 flex items-start justify-between gap-3">
          <div>
            <h4 class="text-base font-semibold text-slate-900">任务详情</h4>
            <p class="mt-1 text-sm app-muted">用于快速判断是设备问题、文档问题，还是参数设置问题。</p>
          </div>
          <UButton variant="ghost" size="xs" icon="i-lucide-x" @click="selectedRecord = null">关闭</UButton>
        </div>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <div class="soft-spot">
            <div class="text-xs font-medium text-slate-500">文件</div>
            <div class="mt-1 text-sm font-semibold text-slate-900">{{ selectedRecord.filename }}</div>
            <div class="mt-1 text-xs app-muted">{{ selectedRecord.username }} · {{ selectedRecord.pages }} 页</div>
          </div>
          <div class="soft-spot">
            <div class="text-xs font-medium text-slate-500">状态</div>
            <div class="mt-1"><UBadge :color="statusColor(selectedRecord.status)" variant="soft" size="xs">{{ statusText(selectedRecord.status) }}</UBadge></div>
            <div class="mt-2 text-xs" :class="statusDetailClass(selectedRecord.status)">{{ selectedRecord.statusDetail || '当前没有额外状态说明。' }}</div>
          </div>
          <div class="soft-spot">
            <div class="text-xs font-medium text-slate-500">时间轴</div>
            <div class="mt-1 text-sm text-slate-900">创建于 {{ formatTime(selectedRecord.createdAt) }}</div>
            <div class="mt-1 text-xs app-muted">最近更新 {{ formatTime(selectedRecord.updatedAt || selectedRecord.createdAt) }}</div>
            <div v-if="['queued', 'processing'].includes(selectedRecord.status)" class="mt-1 text-xs" :class="isRecordStale(selectedRecord) ? 'text-warning' : 'app-muted'">
              已等待 {{ formatStateDuration(selectedRecord.updatedAt || selectedRecord.createdAt) }}
            </div>
          </div>
          <div class="soft-spot">
            <div class="text-xs font-medium text-slate-500">设备与任务</div>
            <div class="mt-1 text-sm text-slate-900">{{ selectedRecord.printerUri }}</div>
            <div class="mt-1 text-xs app-muted">任务 ID：{{ selectedRecord.jobId || '未返回' }}</div>
          </div>
          <div class="soft-spot">
            <div class="text-xs font-medium text-slate-500">打印参数</div>
            <div class="mt-1 text-sm text-slate-900">{{ formatRecordSummary(selectedRecord) }}</div>
            <div class="mt-1 text-xs app-muted">{{ selectedRecord.paperType || '默认纸张类型' }} · {{ selectedRecord.printScaling || '默认缩放' }}</div>
          </div>
          <div class="soft-spot">
            <div class="text-xs font-medium text-slate-500">页面与版式</div>
            <div class="mt-1 text-sm text-slate-900">{{ selectedRecord.pageRange || '全部页面' }}</div>
            <div class="mt-1 text-xs app-muted">{{ selectedRecord.mirror ? '镜像打印已开启' : '镜像打印未开启' }}</div>
          </div>
        </div>
      </div>
    </UCard>

    <UCard class="soft-card">
      <template #header>
        <div>
          <h3 class="flex items-center gap-2 text-xl font-semibold text-slate-900">
            <UIcon name="i-lucide-settings" class="h-5 w-5 text-primary" />
            系统设置
          </h3>
          <p class="mt-1 text-sm app-muted">控制打印记录与文件的自动保留周期。</p>
        </div>
      </template>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 md:items-end">
        <UFormField label="CUPS 地址" hint="例如 cups:631、host.docker.internal:631 或 192.168.1.20:631">
          <UInput v-model="settings.cupsHost" placeholder="输入可访问的 CUPS 地址" class="w-full" />
        </UFormField>

        <UFormField label="自动清理天数" hint="0 表示不自动清理">
          <UInput type="number" step="1" v-model="settings.retentionDays" placeholder="例如 30" class="w-full" />
        </UFormField>

        <div class="flex flex-col gap-2 md:col-span-2">
          <div class="soft-inline-note p-4 text-sm text-slate-700">
            自动清理会删除过期打印记录、对应文件，并执行数据库压缩与 WAL 清理。CUPS 地址保存后会直接替代环境变量。
          </div>
          <div v-if="cupsConnection.message" class="soft-inline-note p-4 text-sm" :class="cupsConnection.ok ? 'text-success' : 'text-error'">
            {{ cupsConnection.message }}
          </div>
          <div class="flex gap-2">
            <UButton color="primary" icon="i-lucide-save" :loading="savingSettings" @click="saveSettings">保存设置</UButton>
            <UButton variant="soft" icon="i-lucide-plug-zap" :loading="testingCUPS" @click="testCUPSConnection">测试连接</UButton>
            <UButton variant="outline" icon="i-lucide-rotate-ccw" @click="loadSettings">重新读取</UButton>
          </div>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

const emit = defineEmits(['logout'])
const toast = useToast()

const users = ref([])
const loadingUsers = ref(false)
const loadingRecords = ref(false)
const savingUser = ref(false)
const savingSettings = ref(false)
const testingCUPS = ref(false)

const form = ref({
  id: null,
  username: '',
  password: '',
  role: 'user',
  protected: false,
  contactName: '',
  phone: '',
  email: ''
})
const userSearch = ref('')

const printFilters = ref({ username: '', start: '', end: '' })
const printRecords = ref([])
const recordSearch = ref('')
const recordStatus = ref('all')
const recordIssueFilter = ref('')
const recordActionState = ref({ id: null, kind: '' })
const selectedRecord = ref(null)

const settings = ref({ retentionDays: '', cupsHost: '' })
const cupsConnection = ref({ ok: false, message: '' })

const isEditing = computed(() => !!form.value.id)
const adminCount = computed(() => users.value.filter(user => user.role === 'admin').length)
const userCount = computed(() => users.value.filter(user => user.role !== 'admin').length)
const queuedPrintCount = computed(() => printRecords.value.filter(record => record.status === 'queued').length)
const processingPrintCount = computed(() => printRecords.value.filter(record => record.status === 'processing').length)
const activePrintCount = computed(() => queuedPrintCount.value + processingPrintCount.value)
const staleActivePrintCount = computed(() => printRecords.value.filter(record => isRecordStale(record)).length)
const failedPrintCount = computed(() => printRecords.value.filter(record => record.status === 'failed').length)
const todayPrintCount = computed(() => {
  const today = new Date()
  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime()
  const end = start + 24 * 60 * 60 * 1000
  return printRecords.value.filter(record => {
    const time = new Date(record.createdAt).getTime()
    return time >= start && time < end
  }).length
})

const filteredUsers = computed(() => {
  const keyword = userSearch.value.trim().toLowerCase()
  if (!keyword) return users.value
  return users.value.filter(user =>
    [user.username, user.contactName, user.email, user.phone]
      .filter(Boolean)
      .some(value => String(value).toLowerCase().includes(keyword))
  )
})

const baseFilteredPrintRecords = computed(() => {
  const keyword = recordSearch.value.trim().toLowerCase()
  return printRecords.value.filter(record => {
    const matchStatus = recordStatus.value === 'all' || record.status === recordStatus.value
    const matchKeyword = !keyword || [
      record.filename,
      record.username,
      record.jobId,
      record.printerUri,
      record.statusDetail
    ]
      .filter(Boolean)
      .some(value => String(value).toLowerCase().includes(keyword))
    return matchStatus && matchKeyword
  })
})
const issueBuckets = computed(() => {
  const buckets = new Map()
  baseFilteredPrintRecords.value.forEach(record => {
    const issue = deriveRecordIssue(record)
    if (!issue) return
    const existing = buckets.get(issue.key)
    if (existing) {
      existing.count += 1
      return
    }
    buckets.set(issue.key, { ...issue, count: 1 })
  })
  return Array.from(buckets.values()).sort((a, b) => b.count - a.count || a.label.localeCompare(b.label, 'zh-CN'))
})
const filteredPrintRecords = computed(() => {
  if (!recordIssueFilter.value) return baseFilteredPrintRecords.value
  return baseFilteredPrintRecords.value.filter(record => deriveRecordIssue(record)?.key === recordIssueFilter.value)
})
const activeIssueFilterLabel = computed(() => issueBuckets.value.find(bucket => bucket.key === recordIssueFilter.value)?.label || '')

const roleItems = [
  { label: '普通用户', value: 'user' },
  { label: '管理员', value: 'admin' }
]

const statusItems = [
  { label: '全部状态', value: 'all' },
  { label: '排队中', value: 'queued' },
  { label: '处理中', value: 'processing' },
  { label: '已打印', value: 'printed' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' }
]

const userColumns = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'username', header: '登录名' },
  { accessorKey: 'role', header: '角色' },
  { accessorKey: 'mustChangePassword', header: '密码状态' },
  { accessorKey: 'contactName', header: '联系人' },
  { accessorKey: 'phone', header: '电话' },
  { accessorKey: 'email', header: '邮箱' },
  { accessorKey: 'createdAt', header: '创建时间' },
  { id: 'actions', header: '操作' }
]

const printColumns = [
  { accessorKey: 'createdAt', header: '时间' },
  { accessorKey: 'username', header: '用户' },
  { accessorKey: 'filename', header: '文件' },
  { accessorKey: 'pages', header: '页数' },
  { accessorKey: 'status', header: '状态' },
  { accessorKey: 'jobId', header: '任务 ID' },
  { id: 'actions', header: '操作' }
]

function getCSRF() {
  const m = document.cookie.match('(^|;)\\s*csrf_token\\s*=\\s*([^;]+)')
  return m ? m.pop() : ''
}

function roleText(role) {
  return role === 'admin' ? '管理员' : '普通用户'
}

function statusText(status) {
  const map = { queued: '排队中', processing: '处理中', printed: '已打印', failed: '失败', cancelled: '已取消' }
  return map[status] || status
}

function statusColor(status) {
  const map = { queued: 'warning', processing: 'primary', printed: 'success', failed: 'error', cancelled: 'neutral' }
  return map[status] || 'neutral'
}

function statusDetailClass(status) {
  if (status === 'failed') return 'text-error'
  if (status === 'cancelled') return 'text-warning'
  return 'app-muted'
}

function summarizeIssueDetail(detail) {
  const text = String(detail || '').trim()
  if (!text) return '失败原因待确认'
  const lower = text.toLowerCase()
  if (lower.includes('timeout') || text.includes('超时')) return '设备响应超时'
  if (lower.includes('unauthorized') || lower.includes('forbidden') || text.includes('认证') || text.includes('权限')) return '权限或认证失败'
  if (lower.includes('refused') || lower.includes('unreachable') || lower.includes('network') || lower.includes('dial tcp') || text.includes('连接')) return '设备连接失败'
  if (lower.includes('busy') || text.includes('繁忙')) return '打印机繁忙'
  if (lower.includes('paper') || lower.includes('media') || text.includes('纸张') || text.includes('纸盒')) return '纸张或纸盒设置不匹配'
  if (lower.includes('format') || lower.includes('pdf') || text.includes('格式') || text.includes('文档')) return '文档格式或内容异常'
  return text.length > 20 ? `${text.slice(0, 20)}…` : text
}

function deriveRecordIssue(record) {
  if (isRecordStale(record)) {
    const label = record.status === 'processing' ? '处理中超时' : '排队超时'
    return { key: `stale:${record.status}`, label, color: 'warning' }
  }
  if (record?.status === 'failed') {
    const label = summarizeIssueDetail(record.statusDetail)
    return { key: `failed:${label}`, label, color: 'error' }
  }
  return null
}

function isRecordStale(record) {
  if (!['queued', 'processing'].includes(record?.status)) return false
  const ref = record.updatedAt || record.createdAt
  if (!ref) return false
  const timestamp = new Date(ref).getTime()
  if (Number.isNaN(timestamp)) return false
  return Date.now() - timestamp >= 10 * 60 * 1000
}

function formatRecordSummary(record) {
  return [
    `${record.copies || 1} 份`,
    record.orientation === 'landscape' ? '横向' : '纵向',
    record.isColor ? '彩色' : '黑白',
    duplexText(record.duplexMode),
    record.paperSize || ''
  ].filter(Boolean).join(' · ')
}

function duplexText(mode) {
  const map = {
    'one-sided': '单面',
    'two-sided-long-edge': '双面长边',
    'two-sided-short-edge': '双面短边'
  }
  return map[mode] || '单面'
}

function formatTime(iso) {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return iso
  }
}

function formatStateDuration(isoStr) {
  if (!isoStr) return '未知'
  const past = new Date(isoStr)
  if (Number.isNaN(past.getTime())) return '未知'
  const diffMs = Date.now() - past.getTime()
  if (diffMs < 0) return '未知'
  const totalSeconds = Math.floor(diffMs / 1000)
  const d = Math.floor(totalSeconds / 86400)
  const h = Math.floor((totalSeconds % 86400) / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  if (d > 0) return `${d}天${h}小时`
  if (h > 0) return `${h}小时${m}分钟`
  if (m > 0) return `${m}分钟`
  return `${totalSeconds}秒`
}

async function readError(resp) {
  try {
    const data = await resp.json()
    return data.error || resp.statusText
  } catch (e) {
    try {
      const text = await resp.text()
      return text || resp.statusText
    } catch {
      return resp.statusText
    }
  }
}

function isRecordActionLoading(id, kind) {
  return recordActionState.value.id === id && recordActionState.value.kind === kind
}

function selectRecord(record) {
  selectedRecord.value = record
}

function toggleIssueFilter(key) {
  recordIssueFilter.value = recordIssueFilter.value === key ? '' : key
}

function clearIssueFilter() {
  recordIssueFilter.value = ''
}

function resetForm() {
  form.value = {
    id: null,
    username: '',
    password: '',
    role: 'user',
    protected: false,
    contactName: '',
    phone: '',
    email: ''
  }
}

function editUser(user) {
  form.value = {
    id: user.id,
    username: user.username,
    password: '',
    role: user.role,
    protected: user.protected,
    contactName: user.contactName || '',
    phone: user.phone || '',
    email: user.email || ''
  }
}

async function loadUsers() {
  loadingUsers.value = true
  try {
    const resp = await fetch('/api/admin/users', { credentials: 'include' })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    users.value = await resp.json()
  } catch (e) {
    toast.add({ title: '加载用户失败', description: e.message, color: 'error' })
  } finally {
    loadingUsers.value = false
  }
}

async function saveUser() {
  if (!form.value.username.trim()) {
    toast.add({ title: '请输入登录名', color: 'warning' })
    return
  }
  if (!isEditing.value && !form.value.password.trim()) {
    toast.add({ title: '新建用户时必须填写密码', color: 'warning' })
    return
  }

  const payload = {
    username: form.value.username.trim(),
    password: form.value.password,
    role: form.value.role,
    contactName: form.value.contactName.trim(),
    phone: form.value.phone.trim(),
    email: form.value.email.trim()
  }
  const url = isEditing.value ? `/api/admin/users/${form.value.id}` : '/api/admin/users'
  const method = isEditing.value ? 'PUT' : 'POST'

  savingUser.value = true
  try {
    const resp = await fetch(url, {
      method,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCSRF()
      },
      body: JSON.stringify(payload)
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    await loadUsers()
    resetForm()
    toast.add({
      title: isEditing.value ? '用户已更新' : '用户已创建',
      description: form.value.password.trim() ? '该用户下次登录时需要先修改密码。' : '',
      color: 'success'
    })
  } catch (e) {
    toast.add({ title: '保存用户失败', description: e.message, color: 'error' })
  } finally {
    savingUser.value = false
  }
}

async function deleteUser(user) {
  if (!confirm(`确认删除用户 ${user.username} ?`)) return
  try {
    const resp = await fetch(`/api/admin/users/${user.id}`, {
      method: 'DELETE',
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    await loadUsers()
    toast.add({ title: '用户已删除', color: 'success' })
  } catch (e) {
    toast.add({ title: '删除用户失败', description: e.message, color: 'error' })
  }
}

async function loadPrintRecords(silent = false) {
  if (!silent) loadingRecords.value = true
  try {
    const params = new URLSearchParams()
    if (printFilters.value.username) params.set('username', printFilters.value.username)
    if (printFilters.value.start) params.set('start', printFilters.value.start)
    if (printFilters.value.end) params.set('end', printFilters.value.end)
    const resp = await fetch(`/api/admin/print-records?${params.toString()}`, { credentials: 'include' })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    const rows = await resp.json()
    printRecords.value = (rows || []).map(record => ({
      ...record,
      copies: record.copies || 1,
      orientation: record.orientation || 'portrait',
      paperSize: record.paperSize || '',
      paperType: record.paperType || '',
      printScaling: record.printScaling || '',
      pageRange: record.pageRange || '',
      mirror: record.mirror === true,
      statusDetail: record.statusDetail || '',
      updatedAt: record.updatedAt || record.createdAt
    }))
  } catch (e) {
    if (!silent) {
      toast.add({ title: '加载打印记录失败', description: e.message, color: 'error' })
    }
  } finally {
    loadingRecords.value = false
  }
}

async function retryRecord(record) {
  recordActionState.value = { id: record.id, kind: 'retry' }
  try {
    const resp = await fetch(`/api/print-records/${record.id}/retry`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    await loadPrintRecords()
    toast.add({ title: '已重新提交打印', color: 'success' })
  } catch (e) {
    toast.add({ title: '重新打印失败', description: e.message, color: 'error' })
  } finally {
    recordActionState.value = { id: null, kind: '' }
  }
}

async function cancelRecord(record) {
  recordActionState.value = { id: record.id, kind: 'cancel' }
  try {
    const resp = await fetch(`/api/print-records/${record.id}/cancel`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    await loadPrintRecords()
    toast.add({ title: '任务已取消', color: 'success' })
  } catch (e) {
    toast.add({ title: '取消任务失败', description: e.message, color: 'error' })
  } finally {
    recordActionState.value = { id: null, kind: '' }
  }
}

async function loadSettings() {
  try {
    const resp = await fetch('/api/admin/settings', { credentials: 'include' })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    const data = await resp.json()
    settings.value.retentionDays = String(data.retentionDays || 0)
    settings.value.cupsHost = data.cupsHost || ''
  } catch (e) {
    toast.add({ title: '加载系统设置失败', description: e.message, color: 'error' })
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    const payload = {
      retentionDays: parseInt(settings.value.retentionDays || '0', 10),
      cupsHost: settings.value.cupsHost.trim()
    }
    const resp = await fetch('/api/admin/settings', {
      method: 'PUT',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCSRF()
      },
      body: JSON.stringify(payload)
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(await readError(resp))
    }
    await loadSettings()
    cupsConnection.value = { ok: false, message: '' }
    toast.add({ title: '系统设置已保存', color: 'success' })
  } catch (e) {
    toast.add({ title: '保存系统设置失败', description: e.message, color: 'error' })
  } finally {
    savingSettings.value = false
  }
}

async function testCUPSConnection() {
  testingCUPS.value = true
  cupsConnection.value = { ok: false, message: '' }
  try {
    const resp = await fetch('/api/setup/test-cups', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cupsHost: settings.value.cupsHost.trim() })
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok || data.ok === false) {
      if (resp.status === 401) emit('logout')
      throw new Error(data.error || resp.statusText)
    }
    cupsConnection.value = {
      ok: true,
      message: `连接正常，发现 ${data.printerCount || 0} 台打印机。`
    }
    toast.add({ title: 'CUPS 连接正常', color: 'success' })
  } catch (e) {
    cupsConnection.value = {
      ok: false,
      message: `连接失败：${e.message}`
    }
    toast.add({ title: 'CUPS 连接失败', description: e.message, color: 'error' })
  } finally {
    testingCUPS.value = false
  }
}

async function refreshAll() {
  await Promise.all([loadUsers(), loadPrintRecords(), loadSettings()])
}

function exportRecordsCSV() {
  const rows = filteredPrintRecords.value
  if (rows.length === 0) {
    toast.add({ title: '没有可导出的记录', color: 'warning' })
    return
  }

  const header = ['时间', '用户', '文件', '页数', '状态', '状态说明', '任务ID', '打印机', '份数', '方向', '双面', '颜色', '纸张', '页码范围']
  const body = rows.map(row => [
    formatTime(row.createdAt),
    row.username,
    row.filename,
    row.pages,
    statusText(row.status),
    row.statusDetail || '',
    row.jobId || '',
    row.printerUri || '',
    row.copies || 1,
    row.orientation === 'landscape' ? '横向' : '纵向',
    duplexText(row.duplexMode),
    row.isColor ? '彩色' : '黑白',
    row.paperSize || '',
    row.pageRange || '全部'
  ])

  const csv = '\ufeff' + [header, ...body]
    .map(line => line.map(value => `"${String(value ?? '').replaceAll('"', '""')}"`).join(','))
    .join('\n')

  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `print-records-${new Date().toISOString().slice(0, 10)}.csv`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
  toast.add({ title: '导出完成', color: 'success' })
}

let recordsTimer = null

function startRecordPolling() {
  if (!recordsTimer) {
    recordsTimer = setInterval(() => {
      void loadPrintRecords(true)
    }, 5000)
  }
}

function stopRecordPolling() {
  clearInterval(recordsTimer)
  recordsTimer = null
}

watch(activePrintCount, count => {
  if (count > 0) {
    startRecordPolling()
    return
  }
  stopRecordPolling()
})

watch(issueBuckets, buckets => {
  if (!recordIssueFilter.value) return
  if (!buckets.some(bucket => bucket.key === recordIssueFilter.value)) {
    recordIssueFilter.value = ''
  }
})

watch(printRecords, rows => {
  if (!selectedRecord.value) return
  selectedRecord.value = rows.find(row => row.id === selectedRecord.value.id) || null
})

onMounted(async () => {
  await refreshAll()
})

onUnmounted(() => {
  stopRecordPolling()
})
</script>
