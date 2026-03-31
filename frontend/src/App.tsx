import React, { useState, useEffect, useRef } from 'react';
import axios from 'axios';
import { 
  TrendingUp, 
  Activity, 
  Zap, 
  Server,
  LayoutDashboard,
  History,
  ShoppingCart
} from 'lucide-react';
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area
} from 'recharts';

// --- Types ---
interface Order {
  id: string;
  symbol: string;
  side: 'buy' | 'sell';
  price: number;
  quantity: number;
  remaining: number;
  status: string;
  created_at: string;
}

interface Trade {
  id: string;
  symbol: string;
  price: number;
  quantity: number;
  buy_order_id: string;
  sell_order_id: string;
  timestamp: string;
}

interface OrderBook {
  bids: Order[];
  asks: Order[];
}

interface MarketMessage {
  type: 'trade' | 'depth';
  symbol: string;
  data: any;
}

// --- Constants ---
const API_BASE = 'http://localhost:8082'; // Matching Engine
const WS_BASE = 'ws://localhost:8083/ws'; // Market Data Service

export default function App() {
  const [symbol] = useState('AAPL');
  const [orderBook, setOrderBook] = useState<OrderBook>({ bids: [], asks: [] });
  const [tradeHistory, setTradeHistory] = useState<Trade[]>([]);
  const [priceData, setPriceData] = useState<{ time: string, price: number }[]>([]);
  const [balance] = useState(10000.0);
  const [userID] = useState('user-1');
  const [latency, setLatency] = useState(0);
  const [wsStatus, setWsStatus] = useState<'connecting' | 'open' | 'closed'>('connecting');

  // Form states
  const [orderSide, setOrderSide] = useState<'buy' | 'sell'>('buy');
  const [orderType, setOrderType] = useState<'limit' | 'market'>('limit');
  const [price, setPrice] = useState('150.00');
  const [quantity, setQuantity] = useState('10');

  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    connectWS();
    fetchHistory();
    return () => ws.current?.close();
  }, [symbol]);

  const connectWS = () => {
    setWsStatus('connecting');
    ws.current = new WebSocket(WS_BASE);

    ws.current.onopen = () => {
      setWsStatus('open');
      console.log('Connected to Market Data WS');
    };

    ws.current.onmessage = (event) => {
      const start = performance.now();
      const msg: MarketMessage = JSON.parse(event.data);
      
      if (msg.type === 'depth') {
        setOrderBook(msg.data);
      } else if (msg.type === 'trade') {
        const newTrade: Trade = msg.data;
        setTradeHistory(prev => [newTrade, ...prev].slice(0, 50));
        setPriceData(prev => [...prev, { 
          time: new Date(newTrade.timestamp).toLocaleTimeString(), 
          price: newTrade.price 
        }].slice(-20));
      }
      setLatency(Math.round(performance.now() - start));
    };

    ws.current.onclose = () => {
      setWsStatus('closed');
      setTimeout(connectWS, 3000); // Reconnect
    };
  };

  const fetchHistory = async () => {
    try {
      const res = await axios.get(`http://localhost:8083/api/market/history`);
      const trades = res.data.reverse();
      setTradeHistory(trades);
      setPriceData(trades.map((t: any) => ({
        time: new Date(t.timestamp).toLocaleTimeString(),
        price: t.price
      })));
    } catch (err) {
      console.error('Failed to fetch history', err);
    }
  };

  const submitOrder = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await axios.post(`${API_BASE}/orders`, {
        user_id: userID,
        symbol: symbol,
        side: orderSide,
        type: orderType,
        price: parseFloat(price),
        quantity: parseInt(quantity)
      });
    } catch (err) {
      console.error('Order submission failed', err);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-200 font-sans p-4 md:p-8">
      {/* Header */}
      <header className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-600 rounded-lg">
            <TrendingUp size={24} className="text-white" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight">Vortex Trading Terminal</h1>
        </div>
        
        <div className="flex items-center gap-6 bg-slate-900 px-6 py-3 rounded-2xl border border-slate-800">
          <div className="flex items-center gap-2">
            <Zap size={16} className={wsStatus === 'open' ? "text-yellow-400" : "text-slate-500"} />
            <span className="text-sm font-medium">{latency}ms Latency</span>
          </div>
          <div className="flex items-center gap-2 border-l border-slate-800 pl-6">
            <Server size={16} className="text-indigo-400" />
            <span className="text-sm font-medium">Nodes: 3 Paxos</span>
          </div>
          <div className="flex items-center gap-2 border-l border-slate-800 pl-6">
            <div className={`w-2 h-2 rounded-full ${wsStatus === 'open' ? 'bg-green-500' : 'bg-red-500'}`}></div>
            <span className="text-sm font-medium uppercase">{wsStatus}</span>
          </div>
        </div>
      </header>

      <main className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column - Market Info & Chart */}
        <div className="lg:col-span-8 space-y-6">
          <div className="bg-slate-900 rounded-2xl p-6 border border-slate-800 shadow-xl">
            <div className="flex justify-between items-center mb-6">
              <div>
                <h2 className="text-3xl font-bold text-white">{symbol}/USD</h2>
                <p className="text-slate-400 text-sm">Real-time price feed via Market Data Service</p>
              </div>
              <div className="text-right">
                <p className="text-2xl font-mono font-bold text-indigo-400">
                  ${priceData.length > 0 ? priceData[priceData.length - 1].price.toFixed(2) : '---'}
                </p>
                <span className="text-xs text-slate-500 font-medium uppercase">Last Trade Price</span>
              </div>
            </div>
            
            <div className="h-[350px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={priceData}>
                  <defs>
                    <linearGradient id="colorPrice" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="#6366f1" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="time" stroke="#64748b" fontSize={12} tickLine={false} axisLine={false} />
                  <YAxis domain={['auto', 'auto']} stroke="#64748b" fontSize={12} tickLine={false} axisLine={false} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '8px' }}
                    itemStyle={{ color: '#818cf8' }}
                  />
                  <Area type="monotone" dataKey="price" stroke="#6366f1" fillOpacity={1} fill="url(#colorPrice)" strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Order Book */}
            <div className="bg-slate-900 rounded-2xl p-6 border border-slate-800 h-[400px] overflow-hidden flex flex-col">
              <div className="flex items-center gap-2 mb-4">
                <LayoutDashboard size={18} className="text-indigo-400" />
                <h3 className="font-bold text-white uppercase tracking-wider text-sm">Order Book Depth</h3>
              </div>
              <div className="grid grid-cols-3 text-xs font-bold text-slate-500 mb-2 px-2">
                <span>PRICE (USD)</span>
                <span className="text-center">SIZE</span>
                <span className="text-right">TOTAL</span>
              </div>
              <div className="flex-1 overflow-y-auto space-y-1 pr-2 custom-scrollbar">
                {/* Asks (Sell) */}
                {[...orderBook.asks].reverse().map((ask, i) => (
                  <div key={i} className="grid grid-cols-3 text-sm py-1 px-2 rounded hover:bg-slate-800 transition-colors bg-red-500/5">
                    <span className="text-red-400 font-mono">${ask.price.toFixed(2)}</span>
                    <span className="text-center font-mono">{ask.remaining}</span>
                    <span className="text-right font-mono">{(ask.price * ask.remaining).toFixed(2)}</span>
                  </div>
                ))}
                
                <div className="py-3 border-y border-slate-800 my-2 text-center bg-slate-950 rounded">
                   <span className="text-indigo-400 font-bold text-lg">
                    ${priceData.length > 0 ? priceData[priceData.length - 1].price.toFixed(2) : '---'}
                   </span>
                </div>

                {/* Bids (Buy) */}
                {orderBook.bids.map((bid, i) => (
                  <div key={i} className="grid grid-cols-3 text-sm py-1 px-2 rounded hover:bg-slate-800 transition-colors bg-green-500/5">
                    <span className="text-green-400 font-mono">${bid.price.toFixed(2)}</span>
                    <span className="text-center font-mono">{bid.remaining}</span>
                    <span className="text-right font-mono">{(bid.price * bid.remaining).toFixed(2)}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Recent Trades */}
            <div className="bg-slate-900 rounded-2xl p-6 border border-slate-800 h-[400px] overflow-hidden flex flex-col">
              <div className="flex items-center gap-2 mb-4">
                <History size={18} className="text-indigo-400" />
                <h3 className="font-bold text-white uppercase tracking-wider text-sm">Recent Settlements</h3>
              </div>
              <div className="grid grid-cols-3 text-xs font-bold text-slate-500 mb-2 px-2">
                <span>PRICE</span>
                <span className="text-center">QUANTITY</span>
                <span className="text-right">TIME</span>
              </div>
              <div className="flex-1 overflow-y-auto space-y-2 pr-2 custom-scrollbar">
                {tradeHistory.map((trade, i) => (
                  <div key={i} className="grid grid-cols-3 text-sm py-2 px-2 border-b border-slate-800/50 hover:bg-slate-800 transition-colors">
                    <span className="font-mono text-white">${trade.price.toFixed(2)}</span>
                    <span className="text-center font-mono text-slate-400">{trade.quantity}</span>
                    <span className="text-right font-mono text-slate-500">{new Date(trade.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit'})}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Right Column - Order Form */}
        <div className="lg:col-span-4 space-y-6">
          <div className="bg-indigo-600 rounded-2xl p-6 shadow-xl shadow-indigo-500/20">
            <h3 className="text-white font-bold text-lg mb-1">Portfolio Value</h3>
            <p className="text-indigo-100 text-sm mb-4 opacity-80">Total available fiat balance</p>
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-bold text-white">${balance.toLocaleString()}</span>
              <span className="text-indigo-200 text-sm font-medium">USD</span>
            </div>
          </div>

          <div className="bg-slate-900 rounded-2xl p-6 border border-slate-800">
            <div className="flex items-center gap-2 mb-6">
              <ShoppingCart size={18} className="text-indigo-400" />
              <h3 className="font-bold text-white uppercase tracking-wider text-sm">Execution Engine</h3>
            </div>

            <form onSubmit={submitOrder} className="space-y-4">
              <div className="grid grid-cols-2 gap-2 p-1 bg-slate-950 rounded-xl border border-slate-800">
                <button 
                  type="button"
                  onClick={() => setOrderSide('buy')}
                  className={`py-2 rounded-lg text-sm font-bold transition-all ${orderSide === 'buy' ? 'bg-green-600 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}
                >
                  BUY
                </button>
                <button 
                  type="button"
                  onClick={() => setOrderSide('sell')}
                  className={`py-2 rounded-lg text-sm font-bold transition-all ${orderSide === 'sell' ? 'bg-red-600 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}
                >
                  SELL
                </button>
              </div>

              <div className="space-y-1">
                <label className="text-[10px] font-bold text-slate-500 uppercase ml-1">Order Type</label>
                <select 
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-indigo-500 transition-colors"
                  value={orderType}
                  onChange={(e) => setOrderType(e.target.value as any)}
                >
                  <option value="limit">Limit Order</option>
                  <option value="market">Market Order</option>
                </select>
              </div>

              <div className="space-y-1">
                <label className="text-[10px] font-bold text-slate-500 uppercase ml-1">Price (USD)</label>
                <div className="relative">
                  <span className="absolute left-4 top-3.5 text-slate-500 text-sm">$</span>
                  <input 
                    type="number"
                    step="0.01"
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-8 pr-4 py-3 text-sm font-mono focus:outline-none focus:border-indigo-500 transition-colors"
                    placeholder="0.00"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    disabled={orderType === 'market'}
                  />
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-[10px] font-bold text-slate-500 uppercase ml-1">Quantity</label>
                <input 
                  type="number"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-sm font-mono focus:outline-none focus:border-indigo-500 transition-colors"
                  placeholder="0"
                  value={quantity}
                  onChange={(e) => setQuantity(e.target.value)}
                />
              </div>

              <div className="pt-4">
                <button 
                  type="submit"
                  className={`w-full py-4 rounded-xl font-bold text-white shadow-lg transition-all transform active:scale-95 ${orderSide === 'buy' ? 'bg-green-600 shadow-green-900/20' : 'bg-red-600 shadow-red-900/20'}`}
                >
                  Place {orderSide.toUpperCase()} Order
                </button>
              </div>
            </form>
          </div>
          
          <div className="p-4 bg-indigo-500/5 rounded-2xl border border-indigo-500/10">
            <div className="flex items-center gap-2 mb-2 text-indigo-400">
              <Activity size={14} />
              <span className="text-[10px] font-bold uppercase tracking-widest">System Consensus</span>
            </div>
            <p className="text-[11px] text-slate-500 leading-relaxed">
              Every trade execution in this terminal undergoes a Paxos-based distributed consensus protocol before final settlement. This ensures fault-tolerance and total order of writes across all ledger nodes.
            </p>
          </div>
        </div>
      </main>
    </div>
  );
}
