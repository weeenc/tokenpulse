<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import AuthShell from '../components/AuthShell.vue';
import { errorMessage } from '../api/client.js';
import { useAuthStore } from '../stores/auth.js';

const form = reactive({ identity: '', password: '' });
const loading = ref(false);
const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
async function submit() {
  loading.value = true;
  try {
    await auth.login(form.identity, form.password);
    await router.push(
      typeof route.query.returnUrl === 'string' ? route.query.returnUrl : '/dashboard',
    );
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}
</script>
<template>
  <AuthShell title="欢迎回来" subtitle="跨设备，掌握每一次 Token 消耗"
    ><el-form label-position="top" @submit.prevent="submit"
      ><el-form-item label="用户名或邮箱"
        ><el-input v-model="form.identity" size="large" autocomplete="username" /></el-form-item
      ><el-form-item label="密码"
        ><el-input
          v-model="form.password"
          size="large"
          type="password"
          show-password
          autocomplete="current-password" /></el-form-item
      ><el-button
        type="primary"
        size="large"
        native-type="submit"
        :loading="loading"
        class="full-button"
        >登录</el-button
      ></el-form
    >
    <p class="auth-switch">
      还没有账户？<router-link :to="{ path: '/register', query: route.query }"
        >创建账户</router-link
      >
    </p></AuthShell
  >
</template>
