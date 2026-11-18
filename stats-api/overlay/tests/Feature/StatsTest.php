<?php

namespace Tests\Feature;

use Illuminate\Support\Facades\Redis;
use Tests\TestCase;

class StatsTest extends TestCase
{
    public function test_stats_returns_json_structure(): void
    {
        Redis::shouldReceive('zcount')->once()->andReturn(3);
        Redis::shouldReceive('zrevrange')->once()->andReturn([
            'Chicken Biryani' => '5',
            'Naan'            => '3',
        ]);

        $this->getJson('/stats')
            ->assertStatus(200)
            ->assertJsonStructure([
                'orders_per_minute',
                'top_dishes' => [['name', 'count']],
            ])
            ->assertJsonPath('orders_per_minute', 3)
            ->assertJsonPath('top_dishes.0.name', 'Chicken Biryani');
    }

    public function test_health_returns_ok_when_redis_is_up(): void
    {
        Redis::shouldReceive('ping')->once()->andReturn(true);

        $this->getJson('/health')
            ->assertStatus(200)
            ->assertJson(['status' => 'ok']);
    }

    public function test_health_returns_503_when_redis_is_down(): void
    {
        Redis::shouldReceive('ping')->once()->andThrow(new \RuntimeException('down'));

        $this->getJson('/health')->assertStatus(503);
    }
}
