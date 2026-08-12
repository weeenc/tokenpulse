<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { buildContributionCalendar, type ContributionPoint } from '../utils/contributions.js';
import { formatTokens } from '../utils/format.js';

interface DetailGroup {
  key: string;
  totalTokens: number;
}

interface DayDetail {
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
  reasoningTokens: number;
  estimatedCostUsd: number;
  messages: number;
  sessions: number;
  events: number;
  sources: DetailGroup[];
  models: DetailGroup[];
}

const props = withDefaults(
  defineProps<{
    points: ContributionPoint[];
    endDate?: Date;
    loading?: boolean;
    detail?: DayDetail | null;
    detailDate?: string | null;
    detailLoading?: boolean;
  }>(),
  {
    endDate: () => new Date(),
    loading: false,
    detail: null,
    detailDate: null,
    detailLoading: false,
  },
);
const emit = defineEmits<{ selectDay: [date: string] }>();

const root = ref<HTMLElement>();
const calendarScroll = ref<HTMLDivElement>();
const tooltip = ref<HTMLElement>();
const view = ref<'2d' | '3d'>('2d');
const calendar = computed(() => buildContributionCalendar(props.points, props.endDate));
const selectedIndex = ref(364);
const previewIndex = ref<number | null>(null);
const activeIndex = computed(() => previewIndex.value ?? selectedIndex.value);
const activeDay = computed(() => calendar.value.days[activeIndex.value] ?? null);
const tooltipOpen = ref(false);
const tooltipPinned = ref(false);
const tooltipStyle = ref({ left: '0px', top: '0px' });
const tooltipDay = computed(() => (tooltipOpen.value ? activeDay.value : null));
const visibleDetail = computed(() =>
  props.detailDate === tooltipDay.value?.date ? props.detail : null,
);
const gridStyle = computed(() => ({ '--calendar-weeks': String(calendar.value.weeks) }));
let inspectionTimer: ReturnType<typeof setTimeout> | undefined;
let tooltipAnchor = { x: 0, aboveY: 0, belowY: 0 };

watch(
  () => props.points,
  () => {
    selectedIndex.value = calendar.value.days.length - 1;
    previewIndex.value = null;
    scrollToLatest();
    closeTooltip();
  },
  { deep: true },
);

watch(view, () => {
  closeTooltip();
  scrollToLatest();
});

watch([visibleDetail, () => props.detailLoading], () => {
  if (tooltipOpen.value) void nextTick(placeTooltip);
});
onMounted(() => {
  scrollToLatest();
  document.addEventListener('pointerdown', closeOnOutsideClick);
  window.addEventListener('resize', closeTooltip);
  window.addEventListener('scroll', closeTooltip, true);
});
onBeforeUnmount(() => {
  clearTimeout(inspectionTimer);
  document.removeEventListener('pointerdown', closeOnOutsideClick);
  window.removeEventListener('resize', closeTooltip);
  window.removeEventListener('scroll', closeTooltip, true);
});

function formatDate(date: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  }).format(new Date(`${date}T12:00:00`));
}

function number(value: number): string {
  return formatTokens(value);
}

function money(value?: number): string {
  if (value == null) return '—';
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(value);
}

function detailValue(key: keyof DayDetail, fallback: number): number {
  const value = visibleDetail.value?.[key];
  return typeof value === 'number' ? value : fallback;
}

function scrollToLatest(): void {
  void nextTick(() => {
    if (calendarScroll.value) calendarScroll.value.scrollLeft = calendarScroll.value.scrollWidth;
  });
}

function previewDay(index: number, event: PointerEvent | FocusEvent): void {
  previewIndex.value = Math.max(0, Math.min(calendar.value.days.length - 1, index));
  tooltipPinned.value = false;
  openTooltip(event.currentTarget as HTMLElement, event instanceof MouseEvent ? event : undefined);
  scheduleInspection(previewIndex.value);
}

function selectDay(index: number, event?: MouseEvent, focus = false): void {
  clearTimeout(inspectionTimer);
  selectedIndex.value = Math.max(0, Math.min(calendar.value.days.length - 1, index));
  previewIndex.value = selectedIndex.value;
  tooltipPinned.value = true;
  const day = calendar.value.days[selectedIndex.value];
  if (day) emit('selectDay', day.date);
  const target =
    (event?.currentTarget as HTMLElement | undefined) ?? dayElement(selectedIndex.value);
  if (target) openTooltip(target, event);
  if (focus) {
    requestAnimationFrame(() => {
      dayElement(selectedIndex.value)?.focus();
    });
  }
}

function leaveDay(): void {
  clearTimeout(inspectionTimer);
  if (tooltipPinned.value) return;
  previewIndex.value = null;
  tooltipOpen.value = false;
}

function scheduleInspection(index: number): void {
  clearTimeout(inspectionTimer);
  inspectionTimer = setTimeout(() => {
    const day = calendar.value.days[index];
    if (tooltipOpen.value && previewIndex.value === index && day) emit('selectDay', day.date);
  }, 140);
}

function dayElement(index: number): HTMLButtonElement | null {
  return root.value?.querySelector<HTMLButtonElement>(`[data-day-index="${index}"]`) ?? null;
}

function openTooltip(target: HTMLElement, pointer?: MouseEvent | PointerEvent): void {
  const rect = target.getBoundingClientRect();
  tooltipAnchor = {
    x: pointer?.clientX ?? rect.left + rect.width / 2,
    aboveY: pointer?.clientY ?? rect.top,
    belowY: pointer?.clientY ?? rect.bottom,
  };
  tooltipOpen.value = true;
  void nextTick(placeTooltip);
}

function placeTooltip(): void {
  if (!tooltip.value) return;
  const rect = tooltip.value.getBoundingClientRect();
  const gutter = 12;
  const left = Math.min(
    window.innerWidth - rect.width - gutter,
    Math.max(gutter, tooltipAnchor.x - rect.width / 2),
  );
  let top = tooltipAnchor.aboveY - rect.height - gutter;
  if (top < gutter)
    top = Math.min(window.innerHeight - rect.height - gutter, tooltipAnchor.belowY + gutter);
  tooltipStyle.value = {
    left: `${Math.round(left)}px`,
    top: `${Math.round(Math.max(gutter, top))}px`,
  };
}

function closeTooltip(): void {
  clearTimeout(inspectionTimer);
  tooltipOpen.value = false;
  tooltipPinned.value = false;
  previewIndex.value = null;
}

function closeOnOutsideClick(event: PointerEvent): void {
  if (!root.value?.contains(event.target as Node)) closeTooltip();
}

function handleKeydown(event: KeyboardEvent, index: number): void {
  const offsets: Record<string, number> = {
    ArrowUp: -1,
    ArrowDown: 1,
    ArrowLeft: -7,
    ArrowRight: 7,
  };
  if (event.key === 'Home') {
    event.preventDefault();
    inspectWithKeyboard(0);
  } else if (event.key === 'End') {
    event.preventDefault();
    inspectWithKeyboard(calendar.value.days.length - 1);
  } else if (event.key in offsets) {
    event.preventDefault();
    inspectWithKeyboard(index + offsets[event.key]);
  } else if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault();
    selectDay(index, undefined, true);
  } else if (event.key === 'Escape') {
    event.preventDefault();
    closeTooltip();
  }
}

function inspectWithKeyboard(index: number): void {
  const targetIndex = Math.max(0, Math.min(calendar.value.days.length - 1, index));
  selectedIndex.value = targetIndex;
  previewIndex.value = targetIndex;
  tooltipPinned.value = false;
  const target = dayElement(targetIndex);
  const day = calendar.value.days[targetIndex];
  if (target) {
    target.focus();
    openTooltip(target);
  }
  if (day) emit('selectDay', day.date);
}
</script>

<template>
  <section ref="root" class="panel contribution-panel" :class="`is-${view}`">
    <div class="contribution-heading">
      <div>
        <div class="contribution-title-row">
          <h2>每日 Token</h2>
          <span>最近一年</span>
        </div>
        <p>每日 Token 使用强度，颜色越亮表示用量越高。</p>
      </div>
      <div class="contribution-actions">
        <div class="view-switch" aria-label="切换贡献图视图" role="group">
          <button
            type="button"
            :class="{ active: view === '2d' }"
            :aria-pressed="view === '2d'"
            @click="view = '2d'"
          >
            2D
          </button>
          <button
            type="button"
            :class="{ active: view === '3d' }"
            :aria-pressed="view === '3d'"
            @click="view = '3d'"
          >
            3D
          </button>
        </div>
        <div class="contribution-stats">
          <strong>{{ calendar.activeDays }} 个活跃日</strong>
          <span>{{ number(calendar.totalTokens) }} tokens</span>
        </div>
      </div>
    </div>

    <div ref="calendarScroll" class="calendar-scroll" :class="{ loading }">
      <div class="calendar-stage" :style="gridStyle">
        <div class="month-row" aria-hidden="true">
          <span
            v-for="month in calendar.months"
            :key="`${month.label}-${month.week}`"
            :style="{ gridColumn: month.week + 1 }"
            >{{ month.label }}</span
          >
        </div>
        <div class="weekday-row" aria-hidden="true">
          <span>一</span><span>三</span><span>五</span>
        </div>
        <div class="contribution-grid" role="grid" aria-label="最近一年的每日 Token 使用">
          <i v-for="index in calendar.leadingDays" :key="`leading-${index}`"></i>
          <button
            v-for="(day, index) in calendar.days"
            :key="day.date"
            type="button"
            role="gridcell"
            class="contribution-cell"
            :class="[`level-${day.level}`, { selected: selectedIndex === index }]"
            :style="{ '--bar-height': `${day.barHeight}px` }"
            :data-day-index="index"
            :aria-label="`${formatDate(day.date)}，${number(day.totalTokens)} tokens`"
            :aria-selected="selectedIndex === index"
            :aria-describedby="
              tooltipOpen && activeIndex === index ? 'contribution-tooltip' : undefined
            "
            :tabindex="selectedIndex === index ? 0 : -1"
            @pointerenter="previewDay(index, $event)"
            @pointerleave="leaveDay"
            @focus="previewDay(index, $event)"
            @blur="leaveDay"
            @click="selectDay(index, $event)"
            @keydown="handleKeydown($event, index)"
          ></button>
          <i v-for="index in calendar.trailingDays" :key="`trailing-${index}`"></i>
        </div>
      </div>
    </div>

    <div class="contribution-legend" aria-label="Token 使用强度由低到高">
      <span>少</span>
      <i v-for="level in [0, 1, 2, 3, 4]" :key="level" :class="`level-${level}`"></i>
      <span>多</span>
    </div>

    <Teleport to="body">
      <Transition name="tooltip-fade">
        <aside
          v-if="tooltipOpen && tooltipDay"
          id="contribution-tooltip"
          ref="tooltip"
          class="contribution-tooltip"
          :class="{ loading: detailLoading && !visibleDetail }"
          :style="tooltipStyle"
          role="tooltip"
        >
          <header>{{ formatDate(tooltipDay.date) }}</header>
          <div class="tooltip-total">
            <span>总 Token</span>
            <strong>{{ number(detailValue('totalTokens', tooltipDay.totalTokens)) }}</strong>
          </div>
          <dl class="tooltip-breakdown">
            <div>
              <dt>Input</dt>
              <dd>{{ number(detailValue('inputTokens', tooltipDay.inputTokens)) }}</dd>
            </div>
            <div>
              <dt>Output</dt>
              <dd>{{ number(detailValue('outputTokens', tooltipDay.outputTokens)) }}</dd>
            </div>
            <div>
              <dt>Cache read</dt>
              <dd>{{ number(detailValue('cachedInputTokens', tooltipDay.cachedInputTokens)) }}</dd>
            </div>
            <div>
              <dt>Reasoning</dt>
              <dd>{{ number(detailValue('reasoningTokens', tooltipDay.reasoningTokens)) }}</dd>
            </div>
            <div>
              <dt>费用</dt>
              <dd>{{ money(visibleDetail?.estimatedCostUsd) }}</dd>
            </div>
            <div>
              <dt>消息</dt>
              <dd>{{ visibleDetail?.messages ?? '—' }}</dd>
            </div>
          </dl>
          <section class="tooltip-groups">
            <h4>客户端</h4>
            <div v-if="visibleDetail?.sources?.length">
              <p v-for="source in visibleDetail.sources.slice(0, 3)" :key="source.key">
                <strong>{{ source.key }}</strong
                ><span>{{ number(source.totalTokens) }}</span>
              </p>
            </div>
            <p v-else class="tooltip-empty">
              {{ detailLoading ? '正在加载详情…' : '无客户端明细' }}
            </p>
          </section>
          <section v-if="visibleDetail?.models?.length" class="tooltip-groups tooltip-models">
            <h4>模型</h4>
            <p v-for="model in visibleDetail.models.slice(0, 3)" :key="model.key">
              <strong>{{ model.key }}</strong
              ><span>{{ number(model.totalTokens) }}</span>
            </p>
          </section>
        </aside>
      </Transition>
    </Teleport>
  </section>
</template>

<style scoped>
.contribution-panel {
  position: relative;
  min-height: 286px;
  margin-bottom: 14px;
  overflow: hidden;
}

.contribution-heading,
.contribution-actions,
.contribution-title-row,
.contribution-legend {
  display: flex;
  align-items: center;
}

.contribution-heading {
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 12px;
}

.contribution-title-row {
  gap: 10px;
}

.contribution-title-row h2 {
  margin: 0;
  color: #dedee2;
  font-size: 15px;
  font-weight: 610;
  letter-spacing: -0.018em;
}

.contribution-title-row span {
  color: #737986;
  font-size: 10px;
}

.contribution-heading p {
  margin: 5px 0 0;
  color: #696e78;
  font-size: 11px;
}

.contribution-actions {
  gap: 16px;
}

.view-switch {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 3px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 9px;
  background: rgba(3, 3, 5, 0.56);
}

.view-switch::before {
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 3px;
  width: calc((100% - 6px) / 2);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.09);
  box-shadow:
    inset 0 1px rgba(255, 255, 255, 0.08),
    0 4px 12px rgba(0, 0, 0, 0.22);
  content: '';
  transform: translateX(0);
  transition: transform 180ms var(--ease-expo);
}

.is-3d .view-switch::before {
  transform: translateX(100%);
}

.view-switch button {
  position: relative;
  z-index: 1;
  min-width: 38px;
  height: 27px;
  padding: 0 9px;
  color: #707681;
  border: 0;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  font-size: 10px;
  font-weight: 620;
  -webkit-tap-highlight-color: transparent;
  transition: color 120ms ease;
}

.view-switch button:hover {
  color: #bec2cb;
}

.view-switch button.active {
  color: #e5e6eb;
}

.view-switch button:focus-visible {
  outline: 1px solid rgba(171, 180, 255, 0.8);
  outline-offset: -2px;
}

.contribution-stats {
  min-width: 116px;
  text-align: right;
}

.contribution-stats strong,
.contribution-stats span {
  display: block;
}

.contribution-stats strong {
  color: #cfd1d7;
  font-size: 11px;
  font-weight: 590;
}

.contribution-stats span {
  margin-top: 4px;
  color: #666c76;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 9px;
}

.calendar-scroll {
  position: relative;
  min-height: 154px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-color: rgba(255, 255, 255, 0.12) transparent;
  scrollbar-width: thin;
  transition: opacity 180ms ease;
}

.calendar-scroll.loading {
  opacity: 0.48;
}

.calendar-stage {
  --cell-size: 11px;
  --cell-gap: 3px;
  position: relative;
  width: calc(var(--calendar-weeks) * (var(--cell-size) + var(--cell-gap)) - var(--cell-gap));
  min-width: 739px;
  margin: 0 auto;
  padding-top: 29px;
}

.month-row {
  position: absolute;
  top: 10px;
  left: 0;
  display: grid;
  width: 100%;
  grid-template-columns: repeat(var(--calendar-weeks), var(--cell-size));
  column-gap: var(--cell-gap);
}

.month-row span {
  color: #5d636d;
  font-size: 8px;
  white-space: nowrap;
}

.weekday-row {
  position: absolute;
  top: 43px;
  left: -21px;
  display: grid;
  height: 67px;
  align-content: space-between;
}

.weekday-row span {
  color: #565c66;
  font-size: 8px;
}

.contribution-grid {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: var(--cell-size);
  grid-template-rows: repeat(7, var(--cell-size));
  gap: var(--cell-gap);
  width: max-content;
  transform-style: preserve-3d;
}

.contribution-grid > i,
.contribution-cell,
.contribution-legend i {
  width: var(--cell-size);
  height: var(--cell-size);
  border-radius: 2.5px;
}

.contribution-grid > i {
  visibility: hidden;
}

.contribution-cell {
  --cell-color: #17181e;
  position: relative;
  display: block;
  padding: 0;
  border: 1px solid rgba(255, 255, 255, 0.035);
  background: var(--cell-color);
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.025);
  cursor: pointer;
  transition:
    transform 160ms var(--ease-expo),
    filter 160ms ease,
    box-shadow 160ms ease;
}

.contribution-cell:hover,
.contribution-cell.selected {
  z-index: 2;
  filter: brightness(1.28);
  box-shadow:
    0 0 0 1px rgba(205, 210, 255, 0.55),
    0 0 13px rgba(94, 106, 210, 0.34);
  transform: scale(1.18);
}

.level-0 {
  --cell-color: #17181e;
  background: var(--cell-color);
}

.level-1 {
  --cell-color: #30375f;
  background: var(--cell-color);
}

.level-2 {
  --cell-color: #4853a0;
  background: var(--cell-color);
}

.level-3 {
  --cell-color: #6572d4;
  background: var(--cell-color);
}

.level-4 {
  --cell-color: #98a3ff;
  background: var(--cell-color);
}

.contribution-legend {
  justify-content: flex-end;
  gap: 4px;
  margin-top: 8px;
}

.contribution-legend span {
  margin: 0 3px;
  color: #5c626c;
  font-size: 8px;
}

.contribution-legend i {
  --cell-size: 9px;
  border: 1px solid rgba(255, 255, 255, 0.04);
}

.is-3d {
  min-height: 463px;
}

.is-3d .calendar-scroll {
  min-height: 325px;
  border: 1px solid rgba(121, 184, 255, 0.07);
  border-radius: 12px;
  background:
    radial-gradient(ellipse 66% 52% at 62% 72%, rgba(31, 111, 235, 0.12), transparent 72%),
    linear-gradient(180deg, rgba(7, 10, 16, 0.56), rgba(8, 12, 20, 0.86));
  box-shadow:
    inset 0 1px rgba(255, 255, 255, 0.025),
    inset 0 -28px 70px rgba(0, 0, 0, 0.18);
}

.is-3d .calendar-stage {
  min-height: 310px;
  padding-top: 105px;
  perspective: 880px;
}

.is-3d .month-row,
.is-3d .weekday-row {
  opacity: 0;
}

.is-3d .contribution-grid {
  margin-left: 10px;
  transform: rotateX(58deg) rotateZ(38deg) scale(0.78);
  transform-origin: 52% 50%;
  transition: transform 260ms var(--ease-expo);
}

.is-3d .level-0 {
  --cell-color: #191f2b;
}

.is-3d .level-1 {
  --cell-color: #4069b2;
}

.is-3d .level-2 {
  --cell-color: #1f6feb;
}

.is-3d .level-3 {
  --cell-color: #388bfd;
}

.is-3d .level-4 {
  --cell-color: #79b8ff;
}

.is-3d .contribution-cell {
  opacity: 1;
  border-color: color-mix(in srgb, var(--cell-color) 78%, white);
  background: var(--cell-color);
  filter: none;
  box-shadow:
    inset 0 1px rgba(255, 255, 255, 0.19),
    0 1px 2px rgba(0, 0, 0, 0.56);
  transform: translateZ(calc(var(--bar-height) + 2px));
  transform-style: preserve-3d;
  transition:
    transform 520ms var(--ease-expo),
    filter 180ms ease,
    box-shadow 180ms ease;
}

.is-3d .contribution-cell::before,
.is-3d .contribution-cell::after {
  position: absolute;
  content: '';
  filter: none;
}

.is-3d .contribution-cell::before {
  top: 100%;
  left: -1px;
  width: calc(100% + 2px);
  height: calc(var(--bar-height) + 2px);
  border-inline: 1px solid rgba(0, 0, 0, 0.34);
  background: color-mix(in srgb, var(--cell-color) 58%, #000);
  box-shadow: inset 0 -1px rgba(0, 0, 0, 0.24);
  transform: rotateX(-90deg);
  transform-origin: top;
}

.is-3d .contribution-cell::after {
  top: -1px;
  right: 100%;
  width: calc(var(--bar-height) + 2px);
  height: calc(100% + 2px);
  border-block: 1px solid rgba(0, 0, 0, 0.3);
  background: color-mix(in srgb, var(--cell-color) 72%, #000);
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.045);
  transform: rotateY(-90deg);
  transform-origin: right;
}

.is-3d .contribution-cell:hover,
.is-3d .contribution-cell.selected {
  opacity: 1;
  filter: none;
  box-shadow:
    inset 0 0 0 1.5px rgba(240, 242, 255, 0.92),
    0 0 18px rgba(56, 139, 253, 0.7);
  transform: translateZ(calc(var(--bar-height) + 8px));
}

.contribution-tooltip {
  position: fixed;
  z-index: 1000;
  width: 280px;
  max-height: calc(100vh - 24px);
  padding: 12px 13px;
  overflow-y: auto;
  color: #d7d9e0;
  border: 1px solid rgba(255, 255, 255, 0.11);
  border-radius: 10px;
  background: rgba(13, 14, 19, 0.97);
  box-shadow:
    0 18px 48px rgba(0, 0, 0, 0.52),
    inset 0 1px rgba(255, 255, 255, 0.045);
  backdrop-filter: blur(18px);
  pointer-events: none;
}

.contribution-tooltip header {
  margin-bottom: 9px;
  color: #858b98;
  font-size: 10px;
  font-weight: 540;
}

.tooltip-total,
.tooltip-breakdown > div,
.tooltip-groups p {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.tooltip-total {
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.075);
}

.tooltip-total span {
  color: #aeb3bf;
  font-size: 11px;
}

.tooltip-total strong {
  color: #f0f1f5;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 17px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

.tooltip-breakdown {
  display: grid;
  gap: 7px;
  margin: 10px 0 0;
}

.tooltip-breakdown dt,
.tooltip-breakdown dd {
  margin: 0;
  font-size: 10px;
}

.tooltip-breakdown dt {
  color: #777d89;
}

.tooltip-breakdown dd {
  color: #c3c7d0;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-variant-numeric: tabular-nums;
}

.tooltip-groups {
  margin-top: 11px;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.075);
}

.tooltip-groups h4 {
  margin: 0 0 7px;
  color: #676d79;
  font-size: 8px;
  font-weight: 650;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.tooltip-groups p {
  min-height: 21px;
  margin: 0;
  font-size: 9px;
}

.tooltip-groups p strong {
  overflow: hidden;
  color: #b3b7c1;
  font-weight: 540;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tooltip-groups p span {
  flex: 0 0 auto;
  color: #737986;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-variant-numeric: tabular-nums;
}

.tooltip-groups .tooltip-empty {
  display: block;
  min-height: 21px;
  color: #5d626d;
}

.tooltip-fade-enter-active,
.tooltip-fade-leave-active {
  transition:
    opacity 120ms ease,
    transform 150ms var(--ease-expo);
}

.tooltip-fade-enter-from,
.tooltip-fade-leave-to {
  opacity: 0;
  transform: translateY(4px) scale(0.98);
}

@media (max-width: 780px) {
  .contribution-heading {
    align-items: flex-start;
  }

  .contribution-actions {
    display: grid;
    gap: 8px;
  }

  .contribution-stats {
    min-width: 90px;
  }

  .calendar-stage {
    margin-inline: 22px 8px;
  }
}

@media (max-width: 560px) {
  .contribution-heading {
    display: block;
  }

  .contribution-actions {
    display: flex;
    justify-content: space-between;
    margin-top: 14px;
  }

  .contribution-tooltip {
    width: min(280px, calc(100vw - 24px));
  }
}

@media (prefers-reduced-motion: reduce) {
  .is-3d .contribution-grid,
  .is-3d .contribution-cell {
    transition: none;
  }
}
</style>
