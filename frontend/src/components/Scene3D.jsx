import { useRef, useMemo } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { Float, MeshDistortMaterial, Sphere, Points, PointMaterial } from '@react-three/drei';
import * as THREE from 'three';

const generateParticles = (count) => {
  const p = new Float32Array(count * 3);
  for (let i = 0; i < count; i++) {
    p[i * 3] = (Math.random() - 0.5) * 15;
    p[i * 3 + 1] = (Math.random() - 0.5) * 15;
    p[i * 3 + 2] = (Math.random() - 0.5) * 15;
  }
  return p;
};

const NeuralCore = () => {
  const coreRef = useRef();

  useFrame((state) => {
    const time = state.clock.getElapsedTime();
    if (coreRef.current) {
      coreRef.current.rotation.z = time * 0.1;
      coreRef.current.rotation.y = time * 0.15;
    }
  });

  return (
    <group ref={coreRef}>
      <Float speed={1.5} rotationIntensity={0.5} floatIntensity={1}>
        <Sphere args={[1.2, 64, 64]}>
          <MeshDistortMaterial
            color="#d97706"
            speed={4}
            distort={0.3}
            radius={1}
            metalness={0.8}
            roughness={0.1}
            emissive="#d97706"
            emissiveIntensity={0.5}
          />
        </Sphere>
      </Float>
      
      {/* Outer Shell */}
      <mesh rotation={[Math.PI / 4, 0, 0]}>
        <torusGeometry args={[2.2, 0.02, 16, 100]} />
        <meshStandardMaterial color="#d97706" emissive="#d97706" emissiveIntensity={2} />
      </mesh>
      <mesh rotation={[-Math.PI / 4, Math.PI / 4, 0]}>
        <torusGeometry args={[2.5, 0.01, 16, 100]} />
        <meshStandardMaterial color="#ffffff" opacity={0.2} transparent />
      </mesh>
    </group>
  );
};

const ParticleField = ({ count = 2000 }) => {
  const points = useMemo(() => generateParticles(count), [count]);

  const pointsRef = useRef();

  useFrame((state) => {
    const time = state.clock.getElapsedTime();
    if (pointsRef.current) {
      pointsRef.current.rotation.y = time * 0.05;
    }
  });

  return (
    <Points ref={pointsRef} positions={points} stride={3}>
      <PointMaterial
        transparent
        color="#d97706"
        size={0.02}
        sizeAttenuation={true}
        depthWrite={false}
        blending={THREE.AdditiveBlending}
      />
    </Points>
  );
};

const Scene3D = () => {
  return (
    <div className="w-full h-[600px] lg:h-[850px] relative pointer-events-none md:pointer-events-auto overflow-visible">
      <Canvas camera={{ position: [0, 0, 8], fov: 45 }}>
        <color attach="background" args={['#050505']} />
        <fog attach="fog" args={['#050505', 5, 20]} />
        
        <ambientLight intensity={0.2} />
        <spotLight position={[10, 10, 10]} angle={0.15} penumbra={1} intensity={2} castShadow />
        <pointLight position={[-10, -10, -10]} intensity={1} color="#d97706" />
        
        <NeuralCore />
        <ParticleField />
        
      </Canvas>
    </div>
  );
};

export default Scene3D;
