<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\Redis;

class StatsController extends Controller
{
    /**
     * GET /stats
     * Live numbers from Redis: orders in the last 60 seconds and the top 5 dishes.
     */
    public function index(): JsonResponse
    {
        return response()->json([
            'orders_per_minute' => $this->ordersPerMinute(),
            'top_dishes'        => $this->topDishes(),
        ]);
    }

    /**
     * GET /health
     */
    public function health(): JsonResponse
    {
        try {
            Redis::ping();
        } catch (\Throwable $e) {
            return response()->json(['status' => 'error', 'detail' => $e->getMessage()], 503);
        }

        return response()->json([
            'status'  => 'ok',
            'version' => getenv('APP_VERSION') ?: 'dev',
        ]);
    }

    private function ordersPerMinute(): int
    {
        $now       = (int) (microtime(true) * 1000);
        $oneMinAgo = $now - 60000;

        // stats:orders_ts is a sorted set, score = order time in ms (written by orders-api)
        return (int) Redis::zcount('stats:orders_ts', $oneMinAgo, $now);
    }

    private function topDishes(): array
    {
        // stats:top_dishes is a sorted set, member = dish name, score = total quantity ordered
        $raw = Redis::zrevrange('stats:top_dishes', 0, 4, ['withscores' => true]);

        $result = [];
        foreach ($raw as $dish => $count) {
            $result[] = ['name' => (string) $dish, 'count' => (int) $count];
        }

        return $result;
    }
}
