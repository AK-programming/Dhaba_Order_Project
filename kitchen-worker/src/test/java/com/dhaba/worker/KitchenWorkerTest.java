package com.dhaba.worker;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.inOrder;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.time.Duration;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.InOrder;
import org.springframework.data.redis.core.ListOperations;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.jdbc.core.JdbcTemplate;

class KitchenWorkerTest {

    private StringRedisTemplate redis;
    private ListOperations<String, String> listOps;
    private ValueOperations<String, String> valueOps;
    private JdbcTemplate jdbc;
    private KitchenWorker worker;

    @SuppressWarnings("unchecked")
    @BeforeEach
    void setUp() {
        redis = mock(StringRedisTemplate.class);
        listOps = mock(ListOperations.class);
        valueOps = mock(ValueOperations.class);
        jdbc = mock(JdbcTemplate.class);
        when(redis.opsForList()).thenReturn(listOps);
        when(redis.opsForValue()).thenReturn(valueOps);
        worker = new KitchenWorker(redis, jdbc, 0);
    }

    @Test
    void doesNothingWhenQueueIsEmpty() {
        when(listOps.rightPop(eq("orders:queue"), any(Duration.class))).thenReturn(null);

        worker.processNextJob();

        verify(jdbc, never()).update(any(String.class), any(Object[].class));
    }

    @Test
    void movesOrderFromCookingToReady() {
        when(listOps.rightPop(eq("orders:queue"), any(Duration.class))).thenReturn("{\"order_id\":42}");
        when(jdbc.update(any(String.class), any(Object[].class))).thenReturn(1);

        worker.processNextJob();

        InOrder order = inOrder(jdbc);
        order.verify(jdbc).update("UPDATE orders SET status = ? WHERE id = ?", "cooking", 42);
        order.verify(jdbc).update("UPDATE orders SET status = ? WHERE id = ?", "ready", 42);
    }

    @Test
    void ignoresMalformedJob() {
        when(listOps.rightPop(eq("orders:queue"), any(Duration.class))).thenReturn("not-json");

        worker.processNextJob();

        verify(jdbc, never()).update(any(String.class), anyInt(), anyInt());
    }

    @Test
    void heartbeatWritesToRedis() {
        worker.writeHeartbeat();

        verify(valueOps).set(eq("kitchen:heartbeat"), any(String.class), any(Duration.class));
    }
}
