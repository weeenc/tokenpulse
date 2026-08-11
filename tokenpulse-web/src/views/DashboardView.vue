<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Monitor, RefreshRight } from '@element-plus/icons-vue';
import { api, data, errorMessage } from '../api/client.js';
import { ElMessage } from 'element-plus';
import TrendChart from '../components/TrendChart.vue';
import { formatTokens, relativeTime } from '../utils/format.js';
import { deviceStatisticsParams, statisticsParams } from '../utils/statistics.js';

interface Totals {
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
  reasoningTokens: number;
}
interface Summary extends Totals {
  today: number;
  week: number;
  month: number;
  estimatedCostUsd: number;
}
interface Group extends Totals {
  key: string;
}
interface Point extends Totals {
  date: string;
}
interface Recent {
  eventId: string;
  source: string;
  model?: string;
  deviceName: string;
  totalTokens: number;
  occurredAt: string;
}
interface DeviceOption {
  id: number;
  deviceName: string;
  status: string;
}
const summary = ref<Summary>({
  today: 0,
  week: 0,
  month: 0,
  totalTokens: 0,
  inputTokens: 0,
  outputTokens: 0,
  cachedInputTokens: 0,
  reasoningTokens: 0,
  estimatedCostUsd: 0,
});
const trend = ref<Point[]>([]);
const sources = ref<Group[]>([]);
const devices = ref<Group[]>([]);
const models = ref<Group[]>([]);
const recent = ref<Recent[]>([]);
const deviceOptions = ref<DeviceOption[]>([]);
const selectedDeviceId = ref<number | null>(null);
const days = ref(30);
const customRange = ref<[Date, Date] | null>(null);
const loading = ref(false);
const timezoneOffsetMinutes = new Date().getTimezoneOffset();
const rangeLabel = computed(() => {
  if (!customRange.value) return `最近 ${days.value} 天`;
  return `${customRange.value[0].toLocaleDateString('zh-CN')} 至 ${customRange.value[1].toLocaleDateString('zh-CN')}`;
});

function localDate(daysAgo = 0): Date {
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  date.setDate(date.getDate() - daysAgo);
  return date;
}

function recentRange(days: number): [Date, Date] {
  return [localDate(days - 1), localDate()];
}

const rangeShortcuts = [
  { text: '今天', value: () => recentRange(1) },
  { text: '昨天', value: () => [localDate(1), localDate(1)] as [Date, Date] },
  { text: '最近 7 天', value: () => recentRange(7) },
  { text: '最近 30 天', value: () => recentRange(30) },
  { text: '最近 90 天', value: () => recentRange(90) },
];

function disableFutureDate(date: Date): boolean {
  return date.getTime() > localDate().getTime();
}

function selectPreset(): void {
  customRange.value = null;
  void load();
}

function selectCustomRange(): void {
  void load();
}

async function load() {
  loading.value = true;
  try {
    const params = statisticsParams(
      days.value,
      customRange.value ?? undefined,
      timezoneOffsetMinutes,
    );
    const deviceParams = deviceStatisticsParams(selectedDeviceId.value);
    [summary.value, trend.value, sources.value, devices.value, models.value, recent.value] =
      (await Promise.all([
        data(
          api.get('/statistics/summary', {
            params: { timezoneOffsetMinutes, ...deviceParams },
          }),
        ),
        data(api.get('/statistics/trend', { params: { ...params, ...deviceParams } })),
        data(api.get('/statistics/by-source', { params: { ...params, ...deviceParams } })),
        data(api.get('/statistics/by-device', { params: { ...params, ...deviceParams } })),
        data(api.get('/statistics/by-model', { params: { ...params, ...deviceParams } })),
        data(api.get('/statistics/recent', { params: deviceParams })),
      ])) as [Summary, Point[], Group[], Group[], Group[], Recent[]];
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}
async function loadDevices() {
  try {
    deviceOptions.value = await data(api.get('/devices'));
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}
onMounted(() => {
  void Promise.all([loadDevices(), load()]);
});
const cards = computed(() => [
  { label: '今日 Token', value: summary.value.today, accent: 'violet' },
  { label: '本周 Token', value: summary.value.week, accent: 'green' },
  { label: '本月 Token', value: summary.value.month, accent: 'amber' },
  { label: '累计 Token', value: summary.value.totalTokens, accent: 'blue' },
]);
const tokenTypes = computed(() => [
  { label: 'Input', value: summary.value.inputTokens },
  { label: 'Output', value: summary.value.outputTokens },
  { label: 'Cached', value: summary.value.cachedInputTokens },
  { label: 'Reasoning', value: summary.value.reasoningTokens },
]);
function number(value: number) {
  return formatTokens(value);
}
function relative(value: string) {
  return relativeTime(value);
}
</script>
<template>
  <div class="page" :aria-busy="loading">
    <header class="page-header">
      <div>
        <span class="eyebrow">OVERVIEW</span>
        <h1>Token 使用概览</h1>
        <p>跨设备追踪你的 AI Coding 消耗。</p>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedDeviceId"
          class="device-filter"
          clearable
          placeholder="全部设备"
          aria-label="按设备筛选"
          @change="load"
        >
          <template #prefix
            ><el-icon><Monitor /></el-icon
          ></template>
          <el-option
            v-for="device in deviceOptions"
            :key="device.id"
            :label="`${device.deviceName}${device.status === 'REVOKED' ? '（已撤销）' : ''}`"
            :value="device.id"
          />
        </el-select>
        <el-button
          :icon="RefreshRight"
          :loading="loading"
          aria-label="刷新数据概览"
          circle
          @click="load"
        />
      </div>
    </header>
    <section class="metric-grid">
      <article
        v-for="card in cards"
        :key="card.label"
        v-spotlight
        class="metric-card"
        :class="card.accent"
      >
        <span>{{ card.label }}</span
        ><strong>{{ number(card.value) }}</strong
        ><small>tokens</small>
      </article>
    </section>
    <section class="type-strip">
      <div v-for="item in tokenTypes" :key="item.label">
        <span>{{ item.label }}</span
        ><strong>{{ number(item.value) }}</strong>
      </div>
    </section>
    <section v-spotlight class="panel chart-panel">
      <div class="panel-heading">
        <div>
          <h2>使用趋势</h2>
          <p>输入、输出与缓存 Token 的每日变化</p>
        </div>
        <div class="range-controls">
          <el-segmented
            v-model="days"
            :options="[
              { label: '7天', value: 7 },
              { label: '30天', value: 30 },
              { label: '90天', value: 90 },
            ]"
            @change="selectPreset"
          />
          <el-date-picker
            v-model="customRange"
            class="trend-range-picker"
            type="daterange"
            format="YYYY/MM/DD"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            aria-label="选择使用趋势的日期范围"
            popper-class="trend-date-picker"
            placement="bottom-end"
            unlink-panels
            clearable
            :disabled-date="disableFutureDate"
            :shortcuts="rangeShortcuts"
            @change="selectCustomRange"
          />
        </div>
      </div>
      <TrendChart :points="trend" />
    </section>
    <section class="split-grid">
      <article v-spotlight class="panel">
        <div class="panel-heading">
          <div>
            <h2>按工具</h2>
            <p>{{ rangeLabel }}</p>
          </div>
        </div>
        <div class="rank-list">
          <div v-for="(item, index) in sources" :key="item.key">
            <span class="rank">{{ index + 1 }}</span
            ><strong>{{ item.key }}</strong>
            <div class="bar">
              <i
                :style="{
                  width: `${Math.max(4, (item.totalTokens / (sources[0]?.totalTokens || 1)) * 100)}%`,
                }"
              ></i>
            </div>
            <b>{{ number(item.totalTokens) }}</b>
          </div>
          <p v-if="!sources.length" class="empty">同步后将在这里显示工具统计</p>
        </div>
      </article>
      <article v-spotlight class="panel">
        <div class="panel-heading">
          <div>
            <h2>按设备</h2>
            <p>你的使用分布</p>
          </div>
          <router-link to="/devices">管理设备</router-link>
        </div>
        <div class="device-bars">
          <div v-for="item in devices" :key="item.key">
            <div>
              <strong>{{ item.key }}</strong
              ><span>{{ number(item.totalTokens) }}</span>
            </div>
            <el-progress
              :percentage="Math.round((item.totalTokens / (devices[0]?.totalTokens || 1)) * 100)"
              :show-text="false"
            />
          </div>
          <p v-if="!devices.length" class="empty">暂无设备数据</p>
        </div>
      </article>
    </section>
    <section class="split-grid lower">
      <article v-spotlight class="panel">
        <div class="panel-heading">
          <div>
            <h2>模型排行</h2>
            <p>动态来源于采集数据</p>
          </div>
        </div>
        <div class="model-rank-list">
          <div v-for="(item, index) in models" :key="item.key">
            <span class="model-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <div class="model-rank-meta">
              <strong>{{ item.key }}</strong>
              <span class="model-bar">
                <i
                  :style="{
                    width: `${Math.max(6, (item.totalTokens / (models[0]?.totalTokens || 1)) * 100)}%`,
                  }"
                ></i>
              </span>
            </div>
            <b>{{ number(item.totalTokens) }}</b>
          </div>
          <p v-if="!models.length" class="empty">同步后将在这里显示模型统计</p>
        </div>
      </article>
      <article v-spotlight class="panel">
        <div class="panel-heading">
          <div>
            <h2>最近同步事件</h2>
            <p>仅展示使用元数据</p>
          </div>
        </div>
        <div class="recent-list">
          <div v-for="item in recent.slice(0, 6)" :key="item.eventId">
            <span class="source-badge">{{ item.source.slice(0, 1).toUpperCase() }}</span>
            <div>
              <strong>{{ item.model || item.source }}</strong
              ><small>{{ item.deviceName }} · {{ relative(item.occurredAt) }}</small>
            </div>
            <b>{{ number(item.totalTokens) }}</b>
          </div>
          <p v-if="!recent.length" class="empty">暂无同步记录</p>
        </div>
      </article>
    </section>
  </div>
</template>
<style scoped>
.header-actions {
  display: flex;
  align-items: center;
  gap: 14px;
}
.device-filter {
  width: 190px;
}
@media (max-width: 780px) {
  .header-actions {
    width: 100%;
    flex-wrap: wrap;
    justify-content: flex-start;
  }
  .device-filter {
    width: min(240px, calc(100% - 48px));
  }
}
</style>
