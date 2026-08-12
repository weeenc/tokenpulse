// @vitest-environment jsdom
import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import ContributionCalendar from '../src/components/ContributionCalendar.vue';

describe('ContributionCalendar', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('shows the same floating day tooltip in 2D and 3D views', async () => {
    const wrapper = mount(ContributionCalendar, {
      attachTo: document.body,
      props: {
        endDate: new Date(2026, 7, 12),
        points: [
          {
            date: '2026-08-11',
            totalTokens: 1_000,
            inputTokens: 700,
            outputTokens: 200,
            cachedInputTokens: 80,
            reasoningTokens: 20,
          },
        ],
        detail: {
          totalTokens: 1_000,
          inputTokens: 700,
          outputTokens: 200,
          cachedInputTokens: 80,
          reasoningTokens: 20,
          estimatedCostUsd: 0.12,
          messages: 4,
          sessions: 2,
          events: 5,
          sources: [{ key: 'codex', totalTokens: 1_000 }],
          models: [{ key: 'gpt-test', totalTokens: 1_000 }],
        },
        detailDate: '2026-08-11',
      },
    });

    const day = wrapper.find('button[aria-label*="2026年8月11日"]');
    await day.trigger('click');

    const tooltip2d = document.querySelector<HTMLElement>('.contribution-tooltip');
    expect(tooltip2d).not.toBeNull();
    expect(tooltip2d?.textContent).toContain('总 Token');
    expect(tooltip2d?.textContent).toContain('费用');
    expect(tooltip2d?.textContent).toContain('codex');
    expect(tooltip2d?.textContent).toContain('gpt-test');
    expect(wrapper.find('.day-detail').exists()).toBe(false);
    expect(wrapper.emitted('selectDay')?.at(-1)).toEqual(['2026-08-11']);

    await wrapper.findAll('.view-switch button')[1].trigger('click');
    expect(wrapper.classes()).toContain('is-3d');
    expect(document.querySelector('.contribution-tooltip')).toBeNull();

    await wrapper.find('button[aria-label*="2026年8月11日"]').trigger('click');
    const tooltip3d = document.querySelector<HTMLElement>('.contribution-tooltip');
    expect(tooltip3d).not.toBeNull();
    expect(tooltip3d?.textContent).toContain('总 Token');
    expect(tooltip3d?.textContent).toContain('gpt-test');
    expect(wrapper.find('button[aria-label*="2026年8月11日"]').classes()).toContain('selected');

    await wrapper.find('button[aria-label*="2026年8月11日"]').trigger('keydown', { key: 'Escape' });
    expect(document.querySelector('.contribution-tooltip')).toBeNull();

    wrapper.unmount();
  });

  it('does not expose zoom controls in the 3D view', async () => {
    const wrapper = mount(ContributionCalendar, {
      props: {
        endDate: new Date(2026, 7, 12),
        points: [],
      },
    });

    expect(wrapper.find('.zoom-control').exists()).toBe(false);
    await wrapper.findAll('.view-switch button')[1].trigger('click');
    expect(wrapper.classes()).toContain('is-3d');
    expect(wrapper.find('.zoom-control').exists()).toBe(false);
    expect(wrapper.get('.calendar-scroll').attributes('title')).toBeUndefined();

    wrapper.unmount();
  });
});
