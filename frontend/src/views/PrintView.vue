<template>
  <div class="page-shell space-y-4 p-1 sm:p-2">
    <section class="hero-surface">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div class="space-y-3">
          <div class="section-kicker">Print Workspace</div>
          <div>
            <h2 class="text-2xl font-semibold text-slate-900 sm:text-3xl">
              {{ props.session?.username ? `${props.session.username}，开始打印` : '打印控制台' }}
            </h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-700">
              上传文档后自动估算页数，保留常用打印配置，同时实时查看设备状态和任务记录。
            </p>
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <div class="status-pill text-xs app-muted">
            <UIcon name="i-lucide-clock-3" class="h-4 w-4 text-primary" />
            {{ requiresPasswordChange ? '完成改密后恢复工作台' : '记录每 5 秒刷新' }}
          </div>
          <UButton variant="outline" size="sm" icon="i-lucide-refresh-cw" @click="refreshAll" :loading="refreshing" :disabled="requiresPasswordChange">
            刷新工作台
          </UButton>
        </div>
      </div>

      <div class="dashboard-grid mt-6">
        <div v-for="card in overviewCards" :key="card.label" class="metric-card">
          <div class="metric-label">{{ card.label }}</div>
          <div class="metric-value">{{ card.value }}</div>
          <div class="metric-helper">{{ card.helper }}</div>
        </div>
      </div>
    </section>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-5">
      <div v-if="!requiresPasswordChange" class="space-y-4 lg:col-span-3">
        <UCard class="soft-card">
          <template #header>
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                  <UIcon name="i-lucide-printer" class="h-4 w-4 text-primary" />
                  打印机选择
                </h3>
                <p class="mt-1 text-sm app-muted">优先保留你最近一次使用的设备。</p>
              </div>
              <div class="status-pill text-xs app-muted">
                在线设备：{{ printers.length }}
              </div>
            </div>
          </template>

          <UAlert
            v-if="!loadingPrinters && printers.length === 0"
            color="warning"
            variant="soft"
            icon="i-lucide-triangle-alert"
            title="暂时没有可用打印机，请确认 CUPS 服务和设备连接正常。"
            class="mb-4"
          />

          <UFormField label="选择打印机" hint="设备列表来自当前 CUPS 实例。">
            <USelect
              v-model="printer"
              :items="printerItems"
              value-key="value"
              label-key="label"
              class="w-full"
              :disabled="loadingPrinters || printers.length === 0"
              @update:model-value="onPrinterChange"
            />
          </UFormField>
        </UCard>

        <UCard class="soft-card">
          <template #header>
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                  <UIcon name="i-lucide-file-up" class="h-4 w-4 text-primary" />
                  文件与页数
                </h3>
                <p class="mt-1 text-sm app-muted">上传后自动估算页数；转换成 PDF 后会再校正一次。</p>
              </div>
              <UButton variant="ghost" size="sm" icon="i-lucide-folder-open" @click="openFilePicker">
                选择文件
              </UButton>
            </div>
          </template>

          <div class="grid gap-3 sm:grid-cols-3">
            <div class="metric-card">
              <div class="metric-label">当前文件</div>
              <div class="metric-value text-lg">{{ currentFileLabel }}</div>
              <div class="metric-helper">{{ selectedFile ? formatFileSize(selectedFile.size) : '支持 PDF / Office / 图片 / 文本' }}</div>
            </div>

            <div class="metric-card">
              <div class="metric-label">预计页数</div>
              <div class="metric-value">{{ pageEstimateLabel }}</div>
              <div class="metric-helper">
                {{ pageEstimateHelper }}
              </div>
            </div>

            <div class="metric-card">
              <div class="metric-label">输出格式</div>
              <div class="metric-value">{{ outputFormatLabel }}</div>
              <div class="metric-helper">{{ converted ? '已生成可打印 PDF 预览。' : '可直接提交原始文件或先转换。' }}</div>
            </div>
          </div>

          <div
            class="soft-dropzone mt-4 cursor-pointer px-4 py-6 text-center transition sm:px-6"
            :class="isDragging ? 'border-primary bg-primary/6 shadow-lg' : 'hover:border-primary/40 hover:bg-white/90'"
            @dragover.prevent="isDragging = true"
            @dragleave="isDragging = false"
            @drop.prevent="onDrop"
            @click="openFilePicker"
          >
            <input ref="fileInput" type="file" class="hidden" @change="onFileChange" />
            <div v-if="!selectedFile">
              <UIcon name="i-lucide-upload-cloud" class="mx-auto mb-3 h-10 w-10 text-primary sm:h-12 sm:w-12" />
              <p class="text-sm font-medium text-slate-900">点击或拖拽文件到这里</p>
              <p class="mt-1 text-xs app-muted">单个文件建议不超过 64 MB，支持 PDF、Word、Excel、PPT、图片和文本。</p>
            </div>
            <div v-else class="flex items-center gap-3 text-left">
              <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <UIcon name="i-lucide-file-check" class="h-6 w-6" />
              </div>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-semibold text-slate-900">{{ selectedFile.name }}</p>
                <p class="mt-1 text-xs app-muted">
                  {{ formatFileSize(selectedFile.size) }} · {{ fileKindLabel(selectedFile) }}
                </p>
              </div>
              <UButton
                variant="ghost"
                size="xs"
                icon="i-lucide-x"
                color="error"
                class="shrink-0"
                @click.stop="clearFile"
              />
            </div>
          </div>

          <div class="mt-4 space-y-3">
            <UAlert v-if="estimatingPages" color="info" variant="soft" icon="i-lucide-loader-circle" title="正在分析文件页数…" />
            <UAlert v-else-if="pageEstimateError" color="warning" variant="soft" icon="i-lucide-file-warning" :title="pageEstimateError" />
            <UAlert v-if="converting" color="info" variant="soft" icon="i-lucide-loader-circle" title="正在转换为 PDF，请稍候…" />
            <UAlert v-if="converted && !converting" color="success" variant="soft" icon="i-lucide-check-circle" title="已生成 PDF，可直接打印或下载预览。" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <UButton
              v-if="canConvert"
              variant="outline"
              icon="i-lucide-file-text"
              :loading="converting"
              @click="convertToPdf"
            >
              转换为 PDF
            </UButton>
            <UButton
              v-if="selectedFile"
              variant="ghost"
              icon="i-lucide-scan-search"
              :loading="estimatingPages"
              @click="estimateCurrentFile"
            >
              重新估算页数
            </UButton>
            <UButton
              v-if="previewUrl"
              variant="ghost"
              icon="i-lucide-download"
              :href="previewUrl"
              :download="downloadName"
              tag="a"
            >
              下载预览
            </UButton>
          </div>
        </UCard>

        <UCard class="soft-card">
          <template #header>
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                  <UIcon name="i-lucide-settings-2" class="h-4 w-4 text-primary" />
                  打印参数
                </h3>
                <p class="mt-1 text-sm app-muted">常用参数会自动记忆，下次打开仍保留。</p>
              </div>

              <div class="flex flex-wrap gap-2">
                <UButton
                  v-for="preset in presetItems"
                  :key="preset.key"
                  size="sm"
                  :variant="isPresetActive(preset.key) ? 'solid' : 'soft'"
                  @click="applyPreset(preset.key)"
                >
                  {{ preset.label }}
                </UButton>
              </div>
            </div>
          </template>

          <div class="space-y-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <UFormField label="颜色模式">
                <div class="soft-segment flex overflow-hidden">
                  <label
                    v-for="item in colorItems"
                    :key="String(item.value)"
                    class="flex flex-1 cursor-pointer items-center justify-center gap-1.5 px-2 py-2 text-sm transition"
                    :class="isColor === item.value ? 'bg-primary text-white font-medium shadow-sm' : 'hover:bg-white/80'"
                  >
                    <input type="radio" :value="item.value" v-model="isColor" class="sr-only" />
                    <UIcon :name="item.icon" class="h-3.5 w-3.5 shrink-0" />
                    <span class="truncate text-xs">{{ item.label }}</span>
                  </label>
                </div>
              </UFormField>

              <UFormField label="打印方向">
                <div class="soft-segment flex overflow-hidden">
                  <label
                    v-for="item in orientationItems"
                    :key="item.value"
                    class="flex flex-1 cursor-pointer items-center justify-center gap-1.5 px-2 py-2 text-sm transition"
                    :class="orientation === item.value ? 'bg-primary text-white font-medium shadow-sm' : 'hover:bg-white/80'"
                  >
                    <input type="radio" :value="item.value" v-model="orientation" class="sr-only" />
                    <UIcon :name="item.icon" class="h-3.5 w-3.5 shrink-0" />
                    <span class="truncate text-xs">{{ item.label }}</span>
                  </label>
                </div>
              </UFormField>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <UFormField label="双面打印">
                <USelect v-model="duplex" :items="duplexItems" value-key="value" label-key="label" class="w-full" />
              </UFormField>

              <UFormField label="份数">
                <UInput v-model.number="copies" type="number" :min="1" :max="99" class="w-full" />
              </UFormField>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <UFormField label="纸张大小">
                <USelect v-model="paperSize" :items="paperSizeItems" value-key="value" label-key="label" class="w-full" />
              </UFormField>

              <UFormField label="纸张类型">
                <USelect v-model="paperType" :items="paperTypeItems" value-key="value" label-key="label" class="w-full" />
              </UFormField>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <UFormField label="缩放">
                <USelect v-model="printScaling" :items="scalingItems" value-key="value" label-key="label" class="w-full" />
              </UFormField>

              <UFormField label="页面范围" :hint="pageRangeError || '如：1-5 8 10-12；留空表示全部页面'">
                <UInput
                  v-model="pageRange"
                  placeholder="留空=全部"
                  class="w-full"
                  :color="pageRangeError ? 'error' : undefined"
                  @input="validatePageRange"
                />
              </UFormField>
            </div>

            <UFormField label="镜像打印">
              <label
                class="soft-spot flex w-fit cursor-pointer items-center gap-2 transition hover:bg-white/90"
                :class="mirror ? 'border-primary bg-primary/5 text-slate-900' : 'text-slate-700'"
              >
                <UCheckbox v-model="mirror" />
                <UIcon name="i-lucide-flip-horizontal" class="h-4 w-4" />
                <span class="text-sm">水平镜像翻转</span>
              </label>
            </UFormField>

            <div class="soft-inline-note p-4 text-sm text-slate-700">
              当前模式：{{ duplexModeText(duplex) }} · {{ isColor ? '彩色' : '黑白' }} · {{ copies }} 份 ·
              {{ orientation === 'portrait' ? '纵向' : '横向' }}
            </div>

            <div class="flex flex-col gap-2 sm:flex-row">
              <UButton
                color="primary"
                size="lg"
                class="flex-1"
                icon="i-lucide-printer"
                :disabled="!canPrint || printing"
                :loading="printing"
                @click="uploadAndPrint"
              >
                提交打印
              </UButton>

              <UButton variant="outline" size="lg" icon="i-lucide-rotate-ccw" @click="resetPrintSettings">
                恢复默认参数
              </UButton>
            </div>
          </div>
        </UCard>

        <UCard v-if="previewUrl || previewType === 'text'" class="soft-card">
          <template #header>
            <div class="flex items-center gap-2 text-lg font-semibold text-slate-900">
              <UIcon name="i-lucide-eye" class="h-4 w-4 text-primary" />
              文件预览
            </div>
          </template>

          <div class="preview-frame">
            <img v-if="previewType === 'image'" :src="previewUrl" alt="preview" class="mx-auto block max-h-96 max-w-full rounded-2xl object-contain p-4" />
            <iframe v-else-if="previewType === 'pdf'" :src="previewUrl" class="w-full" style="height: 540px;" frameborder="0"></iframe>
            <pre v-else-if="previewType === 'text'" class="max-h-72 overflow-auto whitespace-pre-wrap p-4 text-xs leading-6 text-slate-700">{{ textPreview }}</pre>
          </div>
        </UCard>
      </div>

      <div class="space-y-4" :class="requiresPasswordChange ? 'lg:col-span-5' : 'lg:col-span-2'">
        <UCard v-if="!requiresPasswordChange" class="soft-card">
          <template #header>
            <div class="flex flex-col gap-3">
              <div class="flex items-center justify-between">
                <div>
                  <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                    <UIcon name="i-lucide-history" class="h-4 w-4 text-primary" />
                    打印记录
                  </h3>
                  <p class="mt-1 text-sm app-muted">支持按文件名、任务号和状态快速筛选。</p>
                </div>
                <UButton variant="ghost" size="xs" icon="i-lucide-refresh-cw" @click="loadPrintRecords" />
              </div>

              <div class="soft-table-tools">
                <UInput v-model="printRecordSearch" icon="i-lucide-search" placeholder="搜索文件名、任务 ID、打印机" />
                <USelect v-model="recordStatusFilter" :items="recordStatusItems" value-key="value" label-key="label" />
              </div>
            </div>
          </template>

          <div class="mb-3 flex flex-wrap gap-2">
            <div class="status-pill text-xs app-muted">全部 {{ printRecords.length }} 条</div>
            <div class="status-pill text-xs app-muted">排队中 {{ queuedRecordCount }} 条</div>
            <div class="status-pill text-xs app-muted">处理中 {{ processingRecordCount }} 条</div>
            <div class="status-pill text-xs app-muted">关注 {{ staleRecordCount }} 条</div>
            <div class="status-pill text-xs app-muted">失败 {{ failedRecordCount }} 条</div>
          </div>

          <div class="soft-inline-note mb-3 p-3">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <div class="text-sm font-semibold text-slate-900">问题筛选</div>
                <p class="mt-1 text-xs app-muted">把失败原因和长时间无变化的任务压成快捷筛选，方便快速定位异常。</p>
              </div>
              <UButton
                v-if="activeRecordIssueLabel"
                variant="ghost"
                size="xs"
                icon="i-lucide-filter-x"
                @click="clearRecordIssueFilter"
              >
                清除
              </UButton>
            </div>

            <div class="mt-3 flex flex-wrap gap-2">
              <UButton
                v-for="bucket in recordIssueBuckets.slice(0, 6)"
                :key="bucket.key"
                size="xs"
                :variant="recordIssueFilter === bucket.key ? 'solid' : 'soft'"
                :color="bucket.color"
                @click="toggleRecordIssueFilter(bucket.key)"
              >
                {{ bucket.label }} {{ bucket.count }}
              </UButton>
              <div v-if="recordIssueBuckets.length === 0" class="text-xs app-muted">
                当前记录里没有需要重点关注的问题任务。
              </div>
            </div>

            <div v-if="activeRecordIssueLabel" class="mt-2 text-xs app-muted">
              当前问题筛选：{{ activeRecordIssueLabel }}
            </div>
          </div>

          <div class="space-y-2 max-h-[34rem] overflow-y-auto">
            <div v-if="loadingRecords" class="py-6 text-center">
              <UIcon name="i-lucide-loader-circle" class="mx-auto h-5 w-5 animate-spin text-muted" />
            </div>
            <div v-else-if="filteredPrintRecords.length === 0" class="py-6 text-center text-sm app-muted">
              当前筛选条件下没有记录
            </div>
            <div
              v-for="rec in filteredPrintRecords"
              :key="rec.id"
              class="soft-list-item cursor-pointer transition hover:bg-white/90"
              @click="toggleRecord(rec.id)"
            >
              <div class="flex items-start gap-3">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-semibold text-slate-900">{{ rec.filename }}</p>
                  <p class="mt-1 text-xs app-muted">{{ formatPrinterName(rec.printerUri) }} · {{ rec.pages }} 页 · {{ formatTime(rec.createdAt) }}</p>
                  <p class="mt-1 text-[11px] app-muted">最近更新 {{ formatTime(rec.updatedAt || rec.createdAt) }} · {{ formatStateDuration(rec.updatedAt || rec.createdAt) }}前</p>
                  <p v-if="rec.statusDetail" class="mt-2 text-xs" :class="statusDetailClass(rec.status)">{{ rec.statusDetail }}</p>
                  <p v-if="isRecordStale(rec)" class="mt-1 text-xs text-warning">该任务已长时间未变更，建议检查设备或直接取消后重试。</p>
                </div>
                <UBadge :color="statusColor(rec.status)" variant="soft" size="xs">
                  {{ statusText(rec.status) }}
                </UBadge>
              </div>

              <div v-if="expandedRecords.has(rec.id)" class="mt-3 grid grid-cols-1 gap-2 border-t border-slate-200/70 pt-3 text-xs text-slate-600 sm:grid-cols-2">
                <div><span class="font-medium text-slate-800">颜色：</span>{{ rec.isColor ? '彩色' : '黑白' }}</div>
                <div><span class="font-medium text-slate-800">双面：</span>{{ duplexModeText(rec.duplexMode) }}</div>
                <div><span class="font-medium text-slate-800">页数：</span>{{ rec.pages }}</div>
                <div><span class="font-medium text-slate-800">参数：</span>{{ rec.copies }} 份 · {{ orientationText(rec.orientation) }}</div>
                <div><span class="font-medium text-slate-800">最近更新：</span>{{ formatTime(rec.updatedAt || rec.createdAt) }}</div>
                <div v-if="['queued', 'processing'].includes(rec.status)"><span class="font-medium text-slate-800">等待时长：</span>{{ formatStateDuration(rec.updatedAt || rec.createdAt) }}</div>
                <div v-if="rec.statusDetail" class="sm:col-span-2"><span class="font-medium text-slate-800">状态说明：</span>{{ rec.statusDetail }}</div>
                <div v-if="rec.paperSize || rec.paperType"><span class="font-medium text-slate-800">纸张：</span>{{ [paperSizeText(rec.paperSize), paperTypeText(rec.paperType)].filter(Boolean).join(' · ') }}</div>
                <div v-if="rec.printScaling || rec.pageRange || rec.mirror" class="sm:col-span-2"><span class="font-medium text-slate-800">版式：</span>{{ formatRecordLayout(rec) }}</div>
                <div v-if="rec.jobId" class="flex items-center gap-2">
                  <span class="font-medium text-slate-800">任务 ID：</span>
                  <button class="truncate text-left text-primary hover:underline" @click.stop="copyJobId(rec.jobId)">
                    {{ rec.jobId }}
                  </button>
                </div>
                <div class="sm:col-span-2 flex flex-wrap gap-2 pt-1">
                  <UButton
                    size="xs"
                    variant="ghost"
                    icon="i-lucide-download"
                    :href="`/api/print-records/${rec.id}/file`"
                    target="_blank"
                    @click.stop
                  >
                    下载原文件
                  </UButton>
                  <UButton
                    size="xs"
                    variant="ghost"
                    icon="i-lucide-rotate-cw"
                    :loading="isRecordActionLoading(rec.id, 'retry')"
                    @click.stop="retryRecord(rec)"
                  >
                    再打印一次
                  </UButton>
                  <UButton
                    v-if="['queued', 'processing'].includes(rec.status) && rec.jobId"
                    size="xs"
                    variant="outline"
                    color="warning"
                    icon="i-lucide-ban"
                    :loading="isRecordActionLoading(rec.id, 'cancel')"
                    @click.stop="cancelRecord(rec)"
                  >
                    取消任务
                  </UButton>
                </div>
              </div>
            </div>
          </div>
        </UCard>

        <UCard v-if="!requiresPasswordChange" class="soft-card">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                  <UIcon name="i-lucide-activity" class="h-4 w-4 text-primary" />
                  打印机状态
                </h3>
                <p class="mt-1 text-sm app-muted">实时读取设备属性、耗材和队列情况。</p>
              </div>
              <UButton variant="ghost" size="xs" icon="i-lucide-refresh-cw" @click="loadPrinterInfo" :loading="loadingPrinterInfo" />
            </div>
          </template>

          <div>
            <div v-if="!printer" class="py-6 text-center text-sm app-muted">请先选择打印机</div>
            <div v-else-if="loadingPrinterInfo && !printerInfo" class="py-6 text-center">
              <UIcon name="i-lucide-loader-circle" class="mx-auto h-5 w-5 animate-spin text-muted" />
            </div>
            <div v-else-if="printerInfoError" class="py-6 text-center text-sm text-error">
              <UIcon name="i-lucide-wifi-off" class="mx-auto mb-2 h-5 w-5" />
              {{ printerInfoError }}
            </div>
            <div v-else-if="printerInfo" class="space-y-3">
              <div class="soft-spot flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <UIcon name="i-lucide-info" class="h-4 w-4 text-info" />
                  <span class="text-sm font-medium">打印机状态</span>
                </div>
                <UBadge :color="printerStateColor(printerInfo.state)" variant="soft" size="xs">
                  {{ printerStateText(printerInfo.state) }}
                </UBadge>
              </div>

              <div class="soft-spot flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <UIcon name="i-lucide-list-ordered" class="h-4 w-4 text-primary" />
                  <span class="text-sm font-medium">队列任务数</span>
                </div>
                <span class="text-sm font-semibold text-slate-900">{{ printerInfo.queuedJobs }}</span>
              </div>

              <div
                v-if="printerInfo.attributes && printerInfo.attributes['printer-state-change-date-time']"
                class="soft-spot flex items-center justify-between"
              >
                <div class="flex items-center gap-2">
                  <UIcon name="i-lucide-clock" class="h-4 w-4 text-success" />
                  <span class="text-sm font-medium">状态持续</span>
                </div>
                <span class="text-sm">{{ formatStateDuration(printerInfo.attributes['printer-state-change-date-time']) }}</span>
              </div>

              <div v-if="printerInfo.firmwareVersion" class="soft-spot flex items-center justify-between gap-4">
                <div class="flex items-center gap-2">
                  <UIcon name="i-lucide-cpu" class="h-4 w-4 text-secondary" />
                  <span class="text-sm font-medium">固件版本</span>
                </div>
                <span class="max-w-40 truncate text-xs app-muted">{{ printerInfo.firmwareVersion }}</span>
              </div>

              <div v-if="printerInfo.stateMessage" class="soft-inline-note p-3 text-xs text-warning">
                {{ printerInfo.stateMessage }}
              </div>

              <div v-if="printerInfo.markerNames && printerInfo.markerNames.length > 0">
                <div class="mb-2 flex items-center gap-2">
                  <UIcon name="i-lucide-droplets" class="h-4 w-4 text-primary" />
                  <span class="text-sm font-semibold text-slate-900">墨盒信息</span>
                </div>
                <div class="space-y-2">
                  <div v-for="(name, i) in printerInfo.markerNames" :key="`${name}-${i}`" class="soft-spot space-y-2">
                    <div class="flex justify-between text-xs">
                      <span class="text-slate-700">{{ name }}</span>
                      <span :class="markerLevelColor(printerInfo.markerLevels?.[i])">
                        {{ printerInfo.markerLevels?.[i] ?? '?' }}%
                      </span>
                    </div>
                    <div class="h-2 w-full rounded-full bg-slate-200/80">
                      <div
                        class="h-2 rounded-full transition-all"
                        :class="markerBarColor(printerInfo.markerLevels?.[i])"
                        :style="{ width: Math.max(0, Math.min(100, printerInfo.markerLevels?.[i] ?? 0)) + '%' }"
                      ></div>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="printerInfo.mediaReady && printerInfo.mediaReady.length > 0">
                <div class="mb-2 flex items-center gap-2">
                  <UIcon name="i-lucide-layers" class="h-4 w-4 text-secondary" />
                  <span class="text-sm font-semibold text-slate-900">纸盒信息</span>
                </div>
                <div class="space-y-2">
                  <div
                    v-for="(media, i) in printerInfo.mediaReady"
                    :key="`${media}-${i}`"
                    class="soft-spot flex items-center gap-2 text-xs text-slate-700"
                  >
                    <UIcon name="i-lucide-square" class="h-3 w-3 text-muted" />
                    <span>{{ media }}</span>
                  </div>
                </div>
              </div>

              <div v-if="printerInfo.stateReasons && printerInfo.stateReasons.filter(reason => reason !== 'none').length > 0">
                <div class="mb-2 flex items-center gap-2">
                  <UIcon name="i-lucide-alert-triangle" class="h-4 w-4 text-warning" />
                  <span class="text-sm font-semibold text-slate-900">警报</span>
                </div>
                <div class="space-y-2">
                  <div
                    v-for="reason in printerInfo.stateReasons.filter(item => item !== 'none')"
                    :key="reason"
                    class="soft-inline-note px-3 py-2 text-xs text-warning"
                  >
                    {{ reason }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </UCard>

        <UCard v-if="requiresPasswordChange" class="soft-card">
          <template #header>
            <div>
              <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                <UIcon name="i-lucide-shield-alert" class="h-4 w-4 text-warning" />
                需要先更新密码
              </h3>
              <p class="mt-1 text-sm app-muted">当前账户仍在使用初始化密码。改密前，打印、设备查询和后台管理都会被锁定。</p>
            </div>
          </template>

          <div class="grid gap-3 md:grid-cols-3">
            <div class="metric-card">
              <div class="metric-label">当前状态</div>
              <div class="metric-value">受限</div>
              <div class="metric-helper">仅保留会话、登出和改密接口。</div>
            </div>
            <div class="metric-card">
              <div class="metric-label">触发原因</div>
              <div class="metric-value">首次安全收口</div>
              <div class="metric-helper">默认管理员或管理员重置后的密码需要尽快更换。</div>
            </div>
            <div class="metric-card">
              <div class="metric-label">解除方式</div>
              <div class="metric-value">立即改密</div>
              <div class="metric-helper">成功提交后页面会自动恢复完整工作台。</div>
            </div>
          </div>
        </UCard>

        <UCard class="soft-card">
          <template #header>
            <div>
              <h3 class="flex items-center gap-2 text-lg font-semibold text-slate-900">
                <UIcon name="i-lucide-shield-check" class="h-4 w-4 text-primary" />
                账号安全
              </h3>
              <p class="mt-1 text-sm app-muted">支持当前登录用户自助修改密码。</p>
            </div>
          </template>

          <div class="space-y-4">
            <div v-if="isDefaultAdminHint" class="soft-inline-note p-4 text-sm text-slate-700">
              当前登录的是默认管理员账号，建议立即修改密码。
            </div>

            <UFormField label="当前密码" required>
              <UInput v-model="passwordForm.currentPassword" type="password" icon="i-lucide-lock" class="w-full" />
            </UFormField>

            <UFormField label="新密码" hint="至少 6 位字符" required>
              <UInput v-model="passwordForm.newPassword" type="password" icon="i-lucide-key" class="w-full" />
            </UFormField>

            <UFormField label="确认新密码" required>
              <UInput v-model="passwordForm.confirmPassword" type="password" icon="i-lucide-key-round" class="w-full" />
            </UFormField>

            <UButton color="primary" icon="i-lucide-save" :loading="passwordSaving" class="w-full" @click="updatePassword">
              更新密码
            </UButton>
          </div>
        </UCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { jsPDF } from 'jspdf'

const props = defineProps({
  session: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['logout', 'session-refresh'])
const toast = useToast()

const PRINT_PREFERENCES_KEY = 'cups_web_print_preferences_v2'

const printer = ref('')
const printers = ref([])
const loadingPrinters = ref(false)
const printerItems = computed(() =>
  printers.value.map(p => ({ label: `${p.name} — ${p.uri}`, value: p.uri }))
)

const fileInput = ref(null)
const selectedFile = ref(null)
const previewUrl = ref('')
const previewType = ref('')
const textPreview = ref('')
const converting = ref(false)
const converted = ref(false)
const pdfBlob = ref(null)
const downloadName = ref('')
const isDragging = ref(false)
const estimatingPages = ref(false)
const pageEstimate = ref(null)
const pageEstimateError = ref('')

const isColor = ref(true)
const duplex = ref('one-sided')
const orientation = ref('portrait')
const copies = ref(1)
const paperSize = ref('A4')
const paperType = ref('plain')
const printScaling = ref('fit')
const pageRange = ref('')
const pageRangeError = ref('')
const mirror = ref(false)

const printing = ref(false)
const refreshing = ref(false)

const printRecords = ref([])
const loadingRecords = ref(false)
const expandedRecords = ref(new Set())
const printRecordSearch = ref('')
const recordStatusFilter = ref('all')
const recordIssueFilter = ref('')
const recordActionState = ref({ id: null, kind: '' })

const printerInfo = ref(null)
const loadingPrinterInfo = ref(false)
const printerInfoError = ref('')

const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const passwordSaving = ref(false)

const presetItems = [
  { key: 'office', label: '办公文档' },
  { key: 'duplex', label: '省纸双面' },
  { key: 'photo', label: '照片彩印' }
]

const colorItems = [
  { label: '彩色打印', value: true, icon: 'i-lucide-palette' },
  { label: '黑白打印', value: false, icon: 'i-lucide-circle' }
]

const duplexItems = [
  { label: '单面打印', value: 'one-sided' },
  { label: '双面（长边翻页）', value: 'two-sided-long-edge' },
  { label: '双面（短边翻页）', value: 'two-sided-short-edge' }
]

const orientationItems = [
  { label: '纵向', value: 'portrait', icon: 'i-lucide-rectangle-vertical' },
  { label: '横向', value: 'landscape', icon: 'i-lucide-rectangle-horizontal' }
]

const paperSizeItems = [
  { label: 'A4 (210×297mm)', value: 'A4' },
  { label: 'A3 (297×420mm)', value: 'A3' },
  { label: 'A2 (420×594mm)', value: 'A2' },
  { label: 'A1 (594×841mm)', value: 'A1' },
  { label: '5寸 (89×127mm)', value: '5inch' },
  { label: '6寸 (102×152mm)', value: '6inch' },
  { label: '7寸 (127×178mm)', value: '7inch' },
  { label: '8寸 (152×203mm)', value: '8inch' },
  { label: '10寸 (203×254mm)', value: '10inch' },
  { label: 'Letter (8.5×11in)', value: 'Letter' },
  { label: 'Legal (8.5×14in)', value: 'Legal' }
]

const paperTypeItems = [
  { label: '普通纸', value: 'plain' },
  { label: '照片纸', value: 'photo' },
  { label: '光面照片纸', value: 'glossy' },
  { label: '哑光照片纸', value: 'matte' },
  { label: '信封', value: 'envelope' },
  { label: '卡片纸', value: 'cardstock' },
  { label: '标签纸', value: 'labels' },
  { label: '自动选择', value: 'auto' }
]

const scalingItems = [
  { label: '自动', value: 'auto' },
  { label: '自动适应', value: 'auto-fit' },
  { label: '适应纸张', value: 'fit' },
  { label: '填充纸张', value: 'fill' },
  { label: '无缩放', value: 'none' }
]

const recordStatusItems = [
  { label: '全部状态', value: 'all' },
  { label: '排队中', value: 'queued' },
  { label: '处理中', value: 'processing' },
  { label: '已打印', value: 'printed' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' }
]

const selectedPrinterInfo = computed(() => printers.value.find(item => item.uri === printer.value) || null)
const currentFileLabel = computed(() => selectedFile.value?.name || '尚未选择文件')
const pageEstimateLabel = computed(() => {
  if (estimatingPages.value) return '计算中…'
  if (pageEstimate.value) return `${pageEstimate.value.pages} 页`
  return '待检测'
})
const pageEstimateHelper = computed(() => {
  if (pageEstimateError.value) return pageEstimateError.value
  if (pageEstimate.value) {
    return pageEstimate.value.estimated ? '这是估算页数，提交打印后会再校正。' : '已拿到精确页数，可直接提交打印。'
  }
  return '上传文件后自动估算页数。'
})
const outputFormatLabel = computed(() => {
  if (pdfBlob.value || previewType.value === 'pdf') return 'PDF'
  if (previewType.value === 'image') return '图片'
  if (previewType.value === 'text') return '文本 / Office'
  return '待检测'
})
const queuedRecordCount = computed(() => printRecords.value.filter(record => record.status === 'queued').length)
const processingRecordCount = computed(() => printRecords.value.filter(record => record.status === 'processing').length)
const staleRecordCount = computed(() => printRecords.value.filter(record => isRecordStale(record)).length)
const failedRecordCount = computed(() => printRecords.value.filter(record => record.status === 'failed').length)
const baseFilteredPrintRecords = computed(() => {
  const keyword = printRecordSearch.value.trim().toLowerCase()
  return printRecords.value.filter(record => {
    const matchesStatus = recordStatusFilter.value === 'all' || record.status === recordStatusFilter.value
    const matchesKeyword = !keyword || [
      record.filename,
      record.jobId,
      record.printerUri,
      record.statusDetail
    ]
      .filter(Boolean)
      .some(value => String(value).toLowerCase().includes(keyword))
    return matchesStatus && matchesKeyword
  })
})
const recordIssueBuckets = computed(() => {
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
const activeRecordIssueLabel = computed(() => recordIssueBuckets.value.find(bucket => bucket.key === recordIssueFilter.value)?.label || '')
const canPrint = computed(() => !!printer.value && (!!pdfBlob.value || !!selectedFile.value) && !pageRangeError.value)
const canConvert = computed(() => !!selectedFile.value && !converting.value && selectedFile.value.type !== 'application/pdf')
const requiresPasswordChange = computed(() => props.session?.mustChangePassword === true)
const overviewCards = computed(() => {
  if (requiresPasswordChange.value) {
    return [
      {
        label: '安全状态',
        value: '需改密',
        helper: '完成密码更新后，打印与后台功能会自动恢复。'
      },
      {
        label: '当前账号',
        value: props.session?.username || '当前用户',
        helper: props.session?.role === 'admin' ? '管理员权限暂时冻结。' : '普通用户工作台暂时冻结。'
      },
      {
        label: '恢复方式',
        value: '更新密码',
        helper: '使用当前密码和新的安全密码完成修改。'
      },
      {
        label: '影响范围',
        value: '工作台已锁定',
        helper: '设备读取、打印提交和记录查询均会被拦截。'
      }
    ]
  }

  return [
    {
      label: '当前打印机',
      value: selectedPrinterInfo.value?.name || (printers.value.length ? '待选择' : '暂无设备'),
      helper: selectedPrinterInfo.value?.uri || '请检查 CUPS 连接和设备状态。'
    },
    {
      label: '预计页数',
      value: pageEstimateLabel.value,
      helper: pageEstimateHelper.value
    },
    {
      label: '打印模式',
      value: `${duplexModeText(duplex.value)} · ${isColor.value ? '彩色' : '黑白'}`,
      helper: `${copies.value} 份 · ${orientation.value === 'portrait' ? '纵向' : '横向'} · ${paperSize.value}`
    },
      {
        label: '任务历史',
        value: `${printRecords.value.length} 条`,
      helper: `${queuedRecordCount.value} 条排队中，${processingRecordCount.value} 条处理中，${staleRecordCount.value} 条需关注`
      }
    ]
  })
const isDefaultAdminHint = computed(() => props.session?.username === 'admin')

function getCSRF() {
  const m = document.cookie.match('(^|;)\\s*csrf_token\\s*=\\s*([^;]+)')
  return m ? m.pop() : ''
}

function formatFileSize(bytes) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatTime(iso) {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  } catch {
    return iso
  }
}

function formatPrinterName(uri) {
  if (!uri) return ''
  const parts = uri.split('/')
  return parts[parts.length - 1] || uri
}

function fileKindLabel(file) {
  if (!file) return ''
  if (file.type === 'application/pdf') return 'PDF 文档'
  if (file.type.startsWith('image/')) return '图片文件'
  if (file.type.startsWith('text/') || /\.(txt|md|html)$/i.test(file.name)) return '文本文件'
  if (isOfficeFile(file)) return 'Office 文档'
  return '通用文件'
}

function formatStateDuration(isoStr) {
  if (!isoStr) return '未知'
  const past = new Date(isoStr)
  if (isNaN(past.getTime())) return '未知'
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

function isRecordStale(record) {
  if (!['queued', 'processing'].includes(record?.status)) return false
  const ref = record.updatedAt || record.createdAt
  if (!ref) return false
  const timestamp = new Date(ref).getTime()
  if (Number.isNaN(timestamp)) return false
  return Date.now() - timestamp >= 10 * 60 * 1000
}

function statusColor(status) {
  const map = { queued: 'warning', processing: 'primary', printed: 'success', failed: 'error', cancelled: 'neutral' }
  return map[status] || 'neutral'
}

function statusText(status) {
  const map = { queued: '排队中', processing: '处理中', printed: '已打印', failed: '失败', cancelled: '已取消' }
  return map[status] || status
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
  return text.length > 18 ? `${text.slice(0, 18)}…` : text
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

function orientationText(value) {
  return value === 'landscape' ? '横向' : '纵向'
}

function optionLabel(items, value) {
  return items.find(item => item.value === value)?.label || value || ''
}

function paperSizeText(value) {
  return optionLabel(paperSizeItems, value)
}

function paperTypeText(value) {
  return optionLabel(paperTypeItems, value)
}

function printScalingText(value) {
  return optionLabel(scalingItems, value)
}

function formatRecordLayout(record) {
  return [
    record.printScaling ? `缩放 ${printScalingText(record.printScaling)}` : '',
    record.pageRange ? `范围 ${record.pageRange}` : '全部页面',
    record.mirror ? '镜像' : ''
  ].filter(Boolean).join(' · ')
}

function duplexModeText(mode) {
  const map = {
    'one-sided': '单面',
    'two-sided-long-edge': '双面（长边翻页）',
    'two-sided-short-edge': '双面（短边翻页）'
  }
  return map[mode] || map['one-sided']
}

function printerStateColor(state) {
  const map = { idle: 'success', processing: 'warning', stopped: 'error' }
  return map[state] || 'neutral'
}

function printerStateText(state) {
  const map = { idle: '空闲', processing: '打印中', stopped: '已停止' }
  return map[state] || state || '未知'
}

function markerLevelColor(level) {
  if (level === undefined || level === null) return 'text-muted'
  if (level <= 10) return 'text-error font-bold'
  if (level <= 25) return 'text-warning font-medium'
  return 'text-success'
}

function markerBarColor(level) {
  if (level === undefined || level === null) return 'bg-muted'
  if (level <= 10) return 'bg-error'
  if (level <= 25) return 'bg-warning'
  return 'bg-success'
}

function validatePageRange() {
  let val = pageRange.value.trim()
  if (!val) {
    pageRangeError.value = ''
    return
  }

  const normalizedVal = val
    .replace(/[－—–―]/g, '-')
    .replace(/\s*-\s*/g, '-')
    .replace(/[，,]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  if (normalizedVal !== val) {
    pageRange.value = normalizedVal
    val = normalizedVal
  }

  const pattern = /^(\d+(-\d+)?)(\s+\d+(-\d+)?)*$/
  pageRangeError.value = pattern.test(val) ? '' : '格式无效，例如：1-5 8 10-12'
}

function isOfficeFile(file) {
  return /\.(docx?|pptx?|xlsx?)$/i.test(file.name) || [
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'application/msword',
    'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    'application/vnd.ms-powerpoint',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'application/vnd.ms-excel'
  ].includes(file.type)
}

function openFilePicker() {
  fileInput.value?.click()
}

function applyPrintSettings(settings = {}) {
  isColor.value = typeof settings.isColor === 'boolean' ? settings.isColor : true
  duplex.value = duplexItems.some(item => item.value === settings.duplex) ? settings.duplex : 'one-sided'
  orientation.value = orientationItems.some(item => item.value === settings.orientation) ? settings.orientation : 'portrait'
  copies.value = Number.isInteger(settings.copies) && settings.copies > 0 ? settings.copies : 1
  paperSize.value = paperSizeItems.some(item => item.value === settings.paperSize) ? settings.paperSize : 'A4'
  paperType.value = paperTypeItems.some(item => item.value === settings.paperType) ? settings.paperType : 'plain'
  printScaling.value = scalingItems.some(item => item.value === settings.printScaling) ? settings.printScaling : 'fit'
  pageRange.value = typeof settings.pageRange === 'string' ? settings.pageRange : ''
  mirror.value = settings.mirror === true
  validatePageRange()
}

function loadSavedPrintPreferences() {
  try {
    const raw = localStorage.getItem(PRINT_PREFERENCES_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    applyPrintSettings(saved)
    if (typeof saved.printer === 'string') {
      printer.value = saved.printer
    }
  } catch {
    // ignore malformed local storage
  }
}

function savePrintPreferences() {
  const payload = {
    printer: printer.value,
    isColor: isColor.value,
    duplex: duplex.value,
    orientation: orientation.value,
    copies: copies.value,
    paperSize: paperSize.value,
    paperType: paperType.value,
    printScaling: printScaling.value,
    pageRange: pageRange.value,
    mirror: mirror.value
  }
  localStorage.setItem(PRINT_PREFERENCES_KEY, JSON.stringify(payload))
}

function resetPrintSettings(showToast = true) {
  applyPrintSettings()
  if (showToast) {
    toast.add({ title: '已恢复默认打印参数', color: 'success' })
  }
}

function applyPreset(key) {
  if (key === 'office') {
    applyPrintSettings({
      isColor: false,
      duplex: 'one-sided',
      orientation: 'portrait',
      copies: 1,
      paperSize: 'A4',
      paperType: 'plain',
      printScaling: 'fit',
      pageRange: '',
      mirror: false
    })
  } else if (key === 'duplex') {
    applyPrintSettings({
      isColor: false,
      duplex: 'two-sided-long-edge',
      orientation: 'portrait',
      copies: 1,
      paperSize: 'A4',
      paperType: 'plain',
      printScaling: 'fit',
      pageRange: '',
      mirror: false
    })
  } else if (key === 'photo') {
    applyPrintSettings({
      isColor: true,
      duplex: 'one-sided',
      orientation: 'portrait',
      copies: 1,
      paperSize: '6inch',
      paperType: 'photo',
      printScaling: 'fill',
      pageRange: '',
      mirror: false
    })
  }

  toast.add({ title: '已应用预设', description: presetItems.find(item => item.key === key)?.label || '', color: 'success' })
}

function isPresetActive(key) {
  if (key === 'office') {
    return !isColor.value && duplex.value === 'one-sided' && paperSize.value === 'A4' && paperType.value === 'plain'
  }
  if (key === 'duplex') {
    return !isColor.value && duplex.value === 'two-sided-long-edge' && paperSize.value === 'A4' && paperType.value === 'plain'
  }
  if (key === 'photo') {
    return isColor.value && duplex.value === 'one-sided' && paperSize.value === '6inch' && paperType.value === 'photo'
  }
  return false
}

function clearFile() {
  if (previewUrl.value) {
    try {
      URL.revokeObjectURL(previewUrl.value)
    } catch {
      // ignore
    }
  }
  previewUrl.value = ''
  previewType.value = ''
  textPreview.value = ''
  pdfBlob.value = null
  converted.value = false
  selectedFile.value = null
  downloadName.value = ''
  pageEstimate.value = null
  pageEstimateError.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

function onDrop(e) {
  isDragging.value = false
  const file = e.dataTransfer.files[0]
  if (file) processFile(file)
}

function onFileChange(e) {
  const file = e.target.files[0]
  if (file) processFile(file)
}

async function estimatePagesForFile(file, silent = false) {
  if (!file) return
  estimatingPages.value = true
  pageEstimateError.value = ''
  try {
    const fd = new FormData()
    fd.append('file', file, file.name)
    const resp = await fetch('/api/estimate', {
      method: 'POST',
      body: fd,
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error((await resp.json().catch(() => ({}))).error || resp.statusText)
    }
    const data = await resp.json()
    pageEstimate.value = {
      pages: data.pages,
      estimated: !!data.estimated
    }
  } catch (e) {
    pageEstimate.value = null
    pageEstimateError.value = e.message || '页数估算失败'
    if (!silent) {
      toast.add({ title: '页数估算失败', description: pageEstimateError.value, color: 'warning' })
    }
  } finally {
    estimatingPages.value = false
  }
}

async function estimateCurrentFile() {
  const source = pdfBlob.value
    ? new File([pdfBlob.value], downloadName.value || 'document.pdf', { type: 'application/pdf' })
    : selectedFile.value
  if (!source) return
  await estimatePagesForFile(source)
}

function processFile(file) {
  if (file.size > 64 * 1024 * 1024) {
    toast.add({ title: '文件过大', description: '当前前端限制为 64 MB。', color: 'warning' })
    return
  }

  clearFile()
  selectedFile.value = file
  downloadName.value = file.name.replace(/\.[^/.]+$/, '') + '.pdf'

  if (file.type === 'application/pdf') {
    previewUrl.value = URL.createObjectURL(file)
    previewType.value = 'pdf'
    pdfBlob.value = file
    converted.value = true
  } else if (file.type.startsWith('image/')) {
    previewUrl.value = URL.createObjectURL(file)
    previewType.value = 'image'
  } else if (isOfficeFile(file)) {
    previewType.value = 'text'
    textPreview.value = 'Office 文档暂不直接预览，可先转换为 PDF 查看。'
  } else if (file.type.startsWith('text/') || /\.(txt|md|html)$/i.test(file.name)) {
    const reader = new FileReader()
    reader.onload = () => {
      textPreview.value = reader.result
      previewType.value = 'text'
    }
    reader.readAsText(file)
  } else {
    previewType.value = 'text'
    textPreview.value = '此文件类型暂不支持直接预览，但可以直接提交打印。'
  }

  void estimatePagesForFile(file, true)
}

async function imageFileToPdfBlob(file) {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      const canvas = document.createElement('canvas')
      canvas.width = img.width
      canvas.height = img.height
      canvas.getContext('2d').drawImage(img, 0, 0)
      const imgData = canvas.toDataURL('image/jpeg', 1.0)
      const doc = new jsPDF({ unit: 'px', format: [img.width, img.height] })
      doc.addImage(imgData, 'JPEG', 0, 0, img.width, img.height)
      resolve(doc.output('blob'))
    }
    img.onerror = () => reject(new Error('图片加载失败'))
    img.src = URL.createObjectURL(file)
  })
}

function textToPdfBlob(text) {
  const doc = new jsPDF()
  const lines = doc.splitTextToSize(text || '', 180)
  doc.text(lines, 10, 10)
  return doc.output('blob')
}

async function convertOfficeToPdf(file) {
  const fd = new FormData()
  fd.append('file', file, file.name)
  const resp = await fetch('/api/convert', {
    method: 'POST',
    body: fd,
    credentials: 'include',
    headers: { 'X-CSRF-Token': getCSRF() }
  })
  if (!resp.ok) throw new Error('服务端转换失败：' + await resp.text())
  return resp.blob()
}

async function convertToPdf() {
  if (!selectedFile.value) return
  converting.value = true
  try {
    const file = selectedFile.value
    let blob
    if (isOfficeFile(file)) {
      blob = await convertOfficeToPdf(file)
    } else if (file.type.startsWith('image/')) {
      blob = await imageFileToPdfBlob(file)
    } else {
      const text = await file.text()
      blob = textToPdfBlob(text)
    }

    pdfBlob.value = blob
    if (previewUrl.value) {
      try {
        URL.revokeObjectURL(previewUrl.value)
      } catch {
        // ignore
      }
    }
    previewUrl.value = URL.createObjectURL(blob)
    previewType.value = 'pdf'
    converted.value = true

    const pdfFile = new File([blob], downloadName.value || 'document.pdf', { type: 'application/pdf' })
    await estimatePagesForFile(pdfFile, true)

    toast.add({ title: '转换成功', color: 'success', icon: 'i-lucide-check-circle' })
  } catch (e) {
    toast.add({ title: '转换失败', description: e.message, color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    converting.value = false
  }
}

async function uploadAndPrint() {
  if (!printer.value) {
    toast.add({ title: '请选择打印机', color: 'warning' })
    return
  }
  if (pageRangeError.value) {
    toast.add({ title: '页面范围格式有误', color: 'warning' })
    return
  }

  const fileToSend = pdfBlob.value || selectedFile.value
  const filename = pdfBlob.value
    ? (downloadName.value || 'document.pdf')
    : (selectedFile.value ? selectedFile.value.name : 'document')

  const form = new FormData()
  form.append('file', fileToSend, filename)
  form.append('printer', printer.value)
  form.append('duplex', duplex.value)
  form.append('color', isColor.value ? 'true' : 'false')
  form.append('copies', String(copies.value))
  form.append('orientation', orientation.value)
  form.append('paper_size', paperSize.value)
  form.append('paper_type', paperType.value)
  form.append('print_scaling', printScaling.value)
  if (pageRange.value.trim()) form.append('page_range', pageRange.value.trim())
  if (mirror.value) form.append('mirror', 'true')

  printing.value = true
  try {
    const resp = await fetch('/api/print', {
      method: 'POST',
      body: form,
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      const data = await resp.json().catch(() => ({}))
      if (resp.status === 401) emit('logout')
      throw new Error(data.error || resp.statusText)
    }
    const result = await resp.json()
    pageEstimate.value = { pages: result.pages, estimated: false }
    toast.add({
      title: '打印任务已提交',
      description: `任务ID：${result.jobId || '—'}，共 ${result.pages} 页`,
      color: 'success',
      icon: 'i-lucide-check-circle'
    })
    localStorage.setItem('last_printer', printer.value)
    await loadPrintRecords()
  } catch (e) {
    toast.add({ title: '打印失败', description: e.message, color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    printing.value = false
  }
}

async function loadPrintRecords(silent = false) {
  if (!silent) loadingRecords.value = true
  try {
    const resp = await fetch('/api/print-records', { credentials: 'include' })
    if (resp.ok) {
      const data = await resp.json()
      printRecords.value = (data || []).map(record => ({
        id: record.id,
        filename: record.filename,
        printerUri: record.printerUri,
        pages: record.pages,
        status: record.status,
        isColor: record.isColor,
        isDuplex: record.isDuplex,
        duplexMode: record.duplexMode,
        copies: record.copies || 1,
        orientation: record.orientation || 'portrait',
        paperSize: record.paperSize || '',
        paperType: record.paperType || '',
        printScaling: record.printScaling || '',
        pageRange: record.pageRange || '',
        mirror: record.mirror === true,
        jobId: record.jobId,
        statusDetail: record.statusDetail || '',
        createdAt: record.createdAt,
        updatedAt: record.updatedAt || record.createdAt
      }))
    } else if (resp.status === 401) {
      emit('logout')
    }
  } catch (e) {
    if (!silent) {
      toast.add({ title: '加载打印记录失败', description: e.message, color: 'error' })
    }
  } finally {
    loadingRecords.value = false
  }
}

function toggleRecord(id) {
  const next = new Set(expandedRecords.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedRecords.value = next
}

function isRecordActionLoading(id, kind) {
  return recordActionState.value.id === id && recordActionState.value.kind === kind
}

function toggleRecordIssueFilter(key) {
  recordIssueFilter.value = recordIssueFilter.value === key ? '' : key
}

function clearRecordIssueFilter() {
  recordIssueFilter.value = ''
}

async function copyJobId(jobId) {
  if (!jobId) return
  try {
    await navigator.clipboard.writeText(jobId)
    toast.add({ title: '任务 ID 已复制', color: 'success' })
  } catch {
    toast.add({ title: '复制失败', description: '当前浏览器环境不支持剪贴板写入。', color: 'warning' })
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
      const data = await resp.json().catch(() => ({}))
      throw new Error(data.error || resp.statusText)
    }
    const result = await resp.json()
    toast.add({
      title: '已重新提交打印',
      description: `任务 ID：${result.jobId || '—'}`,
      color: 'success'
    })
    await loadPrintRecords()
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
      const data = await resp.json().catch(() => ({}))
      throw new Error(data.error || resp.statusText)
    }
    toast.add({ title: '任务已取消', color: 'success' })
    await loadPrintRecords()
  } catch (e) {
    toast.add({ title: '取消失败', description: e.message, color: 'error' })
  } finally {
    recordActionState.value = { id: null, kind: '' }
  }
}

async function loadPrinterInfo(silent = false) {
  if (!printer.value) return
  if (!silent) loadingPrinterInfo.value = true
  printerInfoError.value = ''
  try {
    const resp = await fetch(`/api/printer-info?uri=${encodeURIComponent(printer.value)}`, { credentials: 'include' })
    if (resp.ok) {
      printerInfo.value = await resp.json()
    } else if (resp.status === 401) {
      emit('logout')
    } else {
      const data = await resp.json().catch(() => ({}))
      printerInfoError.value = data.error || '查询失败'
    }
  } catch {
    printerInfoError.value = '无法连接到打印机'
  } finally {
    loadingPrinterInfo.value = false
  }
}

async function loadPrinters(silent = false) {
  if (!silent) loadingPrinters.value = true
  try {
    const resp = await fetch('/api/printers', { credentials: 'include' })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      throw new Error(resp.statusText)
    }

    printers.value = await resp.json()

    const savedPrinter = printer.value || localStorage.getItem('last_printer')
    if (savedPrinter && printers.value.some(item => item.uri === savedPrinter)) {
      printer.value = savedPrinter
    } else if (!printers.value.some(item => item.uri === printer.value)) {
      printer.value = printers.value[0]?.uri || ''
    }

    if (printer.value) {
      void loadPrinterInfo(true)
    }
  } catch (e) {
    if (!silent) {
      toast.add({ title: '加载打印机失败', description: e.message, color: 'error' })
    }
  } finally {
    loadingPrinters.value = false
  }
}

function onPrinterChange() {
  printerInfo.value = null
  printerInfoError.value = ''
  loadPrinterInfo()
}

async function refreshAll() {
  if (requiresPasswordChange.value) return
  refreshing.value = true
  await Promise.all([loadPrinters(true), loadPrintRecords(true), loadPrinterInfo(true)])
  refreshing.value = false
}

async function updatePassword() {
  if (!passwordForm.value.currentPassword || !passwordForm.value.newPassword || !passwordForm.value.confirmPassword) {
    toast.add({ title: '请填写完整的密码信息', color: 'warning' })
    return
  }
  if (passwordForm.value.newPassword.length < 6) {
    toast.add({ title: '新密码至少需要 6 位字符', color: 'warning' })
    return
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    toast.add({ title: '两次输入的新密码不一致', color: 'warning' })
    return
  }

  passwordSaving.value = true
  try {
    const resp = await fetch('/api/me/password', {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCSRF()
      },
      body: JSON.stringify({
        currentPassword: passwordForm.value.currentPassword,
        newPassword: passwordForm.value.newPassword
      })
    })
    if (!resp.ok) {
      if (resp.status === 401) emit('logout')
      const data = await resp.json().catch(() => ({}))
      throw new Error(data.error || resp.statusText)
    }

    passwordForm.value = {
      currentPassword: '',
      newPassword: '',
      confirmPassword: ''
    }
    toast.add({ title: '密码已更新', color: 'success' })
    emit('session-refresh')
  } catch (e) {
    toast.add({ title: '修改密码失败', description: e.message, color: 'error' })
  } finally {
    passwordSaving.value = false
  }
}

let recordsTimer = null
let printerInfoTimer = null

function startWorkspacePolling() {
  if (!recordsTimer) {
    recordsTimer = setInterval(() => loadPrintRecords(true), 5000)
  }
  if (!printerInfoTimer) {
    printerInfoTimer = setInterval(() => loadPrinterInfo(true), 15000)
  }
}

function stopWorkspacePolling() {
  clearInterval(recordsTimer)
  clearInterval(printerInfoTimer)
  recordsTimer = null
  printerInfoTimer = null
}

async function initializeWorkspace() {
  await loadPrinters()
  await loadPrintRecords()
  startWorkspacePolling()
}

watch([printer, isColor, duplex, orientation, copies, paperSize, paperType, printScaling, pageRange, mirror], () => {
  savePrintPreferences()
  if (printer.value) {
    localStorage.setItem('last_printer', printer.value)
  }
})

watch(recordIssueBuckets, buckets => {
  if (!recordIssueFilter.value) return
  if (!buckets.some(bucket => bucket.key === recordIssueFilter.value)) {
    recordIssueFilter.value = ''
  }
})

watch(requiresPasswordChange, async locked => {
  if (locked) {
    stopWorkspacePolling()
    return
  }
  await initializeWorkspace()
})

onMounted(async () => {
  loadSavedPrintPreferences()
  if (!requiresPasswordChange.value) {
    await initializeWorkspace()
  }
})

onUnmounted(() => {
  stopWorkspacePolling()
  if (previewUrl.value) {
    try {
      URL.revokeObjectURL(previewUrl.value)
    } catch {
      // ignore
    }
  }
})
</script>
