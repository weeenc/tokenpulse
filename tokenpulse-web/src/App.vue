<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Close, DataAnalysis, Monitor, SwitchButton } from '@element-plus/icons-vue';
import { ElConfigProvider } from 'element-plus';
import zhCn from 'element-plus/es/locale/lang/zh-cn';
import BrandMark from './components/BrandMark.vue';
import { useAuthStore } from './stores/auth.js';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const showShell = computed(() => auth.user && !['/login', '/register'].includes(route.path));
const menuOpen = ref(false);
const scrollProgress = ref(0);
const shellStyle = computed(() => ({ '--scroll-progress': scrollProgress.value }));

function updateScroll(): void {
  scrollProgress.value = Math.min(window.scrollY / 480, 1);
}

async function logout() {
  menuOpen.value = false;
  await auth.logout();
  await router.push('/login');
}

watch(
  () => route.path,
  () => {
    menuOpen.value = false;
  },
);
onMounted(() => {
  updateScroll();
  window.addEventListener('scroll', updateScroll, { passive: true });
});
onBeforeUnmount(() => window.removeEventListener('scroll', updateScroll));
</script>

<template>
  <el-config-provider :locale="zhCn">
    <div v-if="showShell" class="app-shell" :class="{ 'menu-open': menuOpen }" :style="shellStyle">
      <div class="ambient-canvas" aria-hidden="true">
        <span class="ambient-blob ambient-blob-primary"></span>
        <span class="ambient-blob ambient-blob-left"></span>
        <span class="ambient-blob ambient-blob-right"></span>
      </div>
      <button
        v-if="menuOpen"
        class="mobile-backdrop"
        aria-label="关闭导航菜单"
        @click="menuOpen = false"
      ></button>
      <aside class="sidebar">
        <div class="sidebar-header">
          <router-link class="brand" to="/dashboard">
            <BrandMark /><span>TokenPulse</span>
          </router-link>
          <button
            class="mobile-menu"
            type="button"
            aria-label="切换导航菜单"
            :aria-expanded="menuOpen"
            @click="menuOpen = !menuOpen"
          >
            <el-icon v-if="menuOpen"><Close /></el-icon>
            <span v-else class="menu-glyph" aria-hidden="true"><i></i></span>
          </button>
        </div>
        <div class="sidebar-panel" :class="{ open: menuOpen }">
          <nav aria-label="主导航">
            <span class="nav-label">Workspace</span>
            <router-link to="/dashboard">
              <el-icon><DataAnalysis /></el-icon><span>数据概览</span><i></i>
            </router-link>
            <router-link to="/devices">
              <el-icon><Monitor /></el-icon><span>设备管理</span><i></i>
            </router-link>
          </nav>
          <div class="sidebar-status">
            <span class="status-dot"></span>
            <div><strong>同步服务在线</strong><small>数据端到端加密传输</small></div>
          </div>
          <div class="profile">
            <div class="avatar">{{ auth.user?.username.slice(0, 1).toUpperCase() }}</div>
            <div>
              <strong>{{ auth.user?.username }}</strong
              ><small>个人账户</small>
            </div>
            <button aria-label="退出" title="退出登录" @click="logout">
              <el-icon><SwitchButton /></el-icon>
            </button>
          </div>
        </div>
      </aside>
      <main class="content"><router-view /></main>
    </div>
    <router-view v-else />
  </el-config-provider>
</template>
