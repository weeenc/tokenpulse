<script setup lang="ts">
import { computed } from 'vue';
import { Moon, Sunny } from '@element-plus/icons-vue';
import { useTheme } from '../utils/theme.js';

const { theme, toggleTheme } = useTheme();
const label = computed(() => `切换到${theme.value === 'dark' ? '亮色' : '深色'}模式`);
</script>

<template>
  <button
    type="button"
    class="theme-toggle"
    :aria-label="label"
    :title="label"
    @click="toggleTheme"
  >
    <component :is="theme === 'dark' ? Sunny : Moon" aria-hidden="true" />
  </button>
</template>
