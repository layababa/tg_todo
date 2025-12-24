import { onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';

export function useSwipeBack() {
  const router = useRouter();
  
  let touchStartX = 0;
  let touchStartY = 0;
  const edgeThreshold = 40; // 离左边缘多少像素内触发
  const swipeThreshold = 60; // 滑动多少像素触发返回

  const handleTouchStart = (e: TouchEvent) => {
    const touch = e.touches[0];
    touchStartX = touch.clientX;
    touchStartY = touch.clientY;
  };

  const handleTouchMove = (e: TouchEvent) => {
    const touch = e.touches[0];
    const deltaX = touch.clientX - touchStartX;
    const deltaY = Math.abs(touch.clientY - touchStartY);
    const screenWidth = window.innerWidth;

    const isLeftEdge = touchStartX <= edgeThreshold;
    const isRightEdge = touchStartX >= screenWidth - edgeThreshold;

    // 如果检测到是从边缘开始的水平滑动，
    // 我们强制拦截该事件，防止触发 Telegram Mini App 的默认逻辑
    if ((isLeftEdge && deltaX > 10 && deltaX > deltaY) || 
        (isRightEdge && deltaX < -10 && Math.abs(deltaX) > deltaY)) {
      if (e.cancelable) e.preventDefault();
    }
  };

  const handleTouchEnd = (e: TouchEvent) => {
    const touch = e.changedTouches[0];
    const deltaX = touch.clientX - touchStartX;
    const deltaY = Math.abs(touch.clientY - touchStartY);
    const screenWidth = window.innerWidth;

    const isLeftSwipe = touchStartX <= edgeThreshold && deltaX > swipeThreshold;
    const isRightSwipe = touchStartX >= screenWidth - edgeThreshold && deltaX < -swipeThreshold;

    // 1. 必须从左边缘开始向右划，或从右边缘开始向左划
    // 2. 水平滑动距离超过阈值
    // 3. 水平滑动明显大于垂直滑动 (防止误触滚动)
    // 4. 只有在可以返回时触发
    if (
      (isLeftSwipe || isRightSwipe) && 
      Math.abs(deltaX) > deltaY * 1.5 &&
      window.history.length > 1
    ) {
      router.back();
    }
  };

  onMounted(() => {
    window.addEventListener('touchstart', handleTouchStart, { passive: true });
    window.addEventListener('touchmove', handleTouchMove, { passive: false });
    window.addEventListener('touchend', handleTouchEnd, { passive: true });
  });

  onUnmounted(() => {
    window.removeEventListener('touchstart', handleTouchStart);
    window.removeEventListener('touchmove', handleTouchMove);
    window.removeEventListener('touchend', handleTouchEnd);
  });
}

