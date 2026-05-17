import { Play, Pause, Clock } from 'lucide-react';
import { motion } from 'framer-motion';

const GlobalPlaybackBar = ({ 
    currentTime, 
    duration, 
    onTimeChange, 
    isPlaying, 
    onTogglePlay 
}) => {
    return (
        <motion.div 
            initial={{ y: 100 }}
            animate={{ y: 0 }}
            className="absolute bottom-8 left-1/2 -translate-x-1/2 w-full max-w-2xl bg-slate-900/80 backdrop-blur-2xl border border-white/10 rounded-3xl p-4 shadow-2xl z-[60] flex items-center gap-6"
        >
            <button onClick={onTogglePlay} className="p-4 bg-accent-orange rounded-2xl text-white shadow-lg shadow-orange-500/20">
                {isPlaying ? <Pause size={20} fill="currentColor" /> : <Play size={20} fill="currentColor" />}
            </button>
            
            <div className="flex-1 space-y-2">
                <div className="flex justify-between items-center text-[10px] font-black uppercase tracking-widest text-slate-500">
                    <div className="flex items-center gap-2"><Clock size={12}/> Time-Travel</div>
                    <div>{currentTime.toFixed(2)}s / {duration.toFixed(2)}s</div>
                </div>
                <input 
                    type="range"
                    min="0"
                    max={duration}
                    step="0.01"
                    value={currentTime}
                    onChange={(e) => onTimeChange(parseFloat(e.target.value))}
                    className="w-full h-1.5 bg-white/10 rounded-lg appearance-none cursor-pointer accent-accent-orange"
                />
            </div>
        </motion.div>
    );
};

export default GlobalPlaybackBar;
