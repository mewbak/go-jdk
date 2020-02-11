package irgen

import (
	"fmt"
	"math"

	"github.com/quasilyte/GopherJRE/bytecode"
	"github.com/quasilyte/GopherJRE/ir"
	"github.com/quasilyte/GopherJRE/jclass"
)

type generator struct {
	f         *jclass.File
	m         *jclass.Method
	tmpOffset int64

	tmp       int64
	freelist  []int64
	st        operandStack
	out       []ir.Inst
	toResolve []unresolvedBranch
}

type unresolvedBranch struct {
	pc     int32
	branch *int64
}

func (g *generator) ConvertMethod(f *jclass.File, m *jclass.Method) []ir.Inst {
	g.reset(f, m)
	return g.convert()
}

func (g *generator) reset(f *jclass.File, m *jclass.Method) {
	g.f = f
	g.m = m
	g.tmp = 0
	g.freelist = g.freelist[:0]
	g.st.reset()
	g.out = g.out[:0]
	g.toResolve = g.toResolve[:0]
}

func (g *generator) nextTmp() int64 {
	if len(g.freelist) != 0 {
		i := g.freelist[len(g.freelist)-1]
		g.freelist = g.freelist[:len(g.freelist)-1]
		return i
	}
	v := g.tmp
	g.tmp++
	return v
}

func (g *generator) drop(n int) {
	for i := 0; i < n; i++ {
		v := g.st.get(i)
		if v.kind == valueTmp {
			g.freelist = append(g.freelist, v.value)
		}
	}
	g.st.drop(n)
}

func (g *generator) irArg(n int) ir.Arg {
	v := g.st.get(n)
	switch v.kind {
	case valueIntLocal, valueLongLocal, valueFloatLocal, valueDoubleLocal:
		return ir.Arg{Kind: ir.ArgReg, Value: v.value}
	case valueTmp:
		return ir.Arg{Kind: ir.ArgReg, Value: v.value + g.tmpOffset}
	case valueIntConst:
		return ir.Arg{Kind: ir.ArgIntConst, Value: v.value}
	case valueLongConst:
		return ir.Arg{Kind: ir.ArgLongConst, Value: v.value}
	case valueFloatConst:
		return ir.Arg{Kind: ir.ArgFloatConst, Value: v.value}
	case valueDoubleConst:
		return ir.Arg{Kind: ir.ArgDoubleConst, Value: v.value}
	default:
		panic(fmt.Sprintf("can't convert arg %#v", v))
	}
}

func (g *generator) convert() []ir.Inst {
	var code []byte
	for _, attr := range g.m.Attrs {
		if attr, ok := attr.(jclass.CodeAttribute); ok {
			code = attr.Code
			g.tmpOffset = int64(attr.MaxLocals)
			break
		}
	}

	pc2index := make(map[int32]int32, len(code)/2)

	pc := 0
	for pc < len(code) {
		pc2index[int32(pc)] = int32(len(g.out))
		op := bytecode.Op(code[pc])

		switch op {
		case bytecode.Iconstm1:
			g.st.push(valueIntConst, -1)
		case bytecode.Iconst0, bytecode.Iconst1, bytecode.Iconst2, bytecode.Iconst3, bytecode.Iconst4, bytecode.Iconst5:
			g.st.push(valueIntConst, int64(op-bytecode.Iconst0))
		case bytecode.Fconst0:
			g.st.push(valueFloatConst, int64(math.Float32bits(0.0)))
		case bytecode.Fconst1:
			g.st.push(valueFloatConst, int64(math.Float32bits(1.0)))
		case bytecode.Fconst2:
			g.st.push(valueFloatConst, int64(math.Float32bits(2.0)))
		case bytecode.Dconst0:
			g.st.push(valueDoubleConst, int64(math.Float64bits(0.0)))
		case bytecode.Dconst1:
			g.st.push(valueDoubleConst, int64(math.Float64bits(1.0)))

		case bytecode.Bipush:
			ib := code[pc+1]
			g.st.push(valueIntConst, int64(ib))
		case bytecode.Sipush:
			ib1 := int16(code[pc+1])
			ib2 := int16(code[pc+2])
			g.st.push(valueIntConst, int64(ib1<<8+ib2))

		case bytecode.Ldc:
			i := uint(code[pc+1])
			switch c := g.f.Consts[i]; c := c.(type) {
			case *jclass.IntConst:
				g.st.push(valueIntConst, int64(c.Value))
			case *jclass.FloatConst:
				g.st.push(valueFloatConst, int64(math.Float32bits(c.Value)))
			default:
				panic(fmt.Sprintf("%T const ldc", c))
			}
		case bytecode.Ldc2w:
			ib1 := uint(code[pc+1])
			ib2 := uint(code[pc+2])
			i := ib1<<8 + ib2
			switch c := g.f.Consts[i]; c := c.(type) {
			case *jclass.IntConst:
				g.st.push(valueIntConst, int64(c.Value))
			case *jclass.LongConst:
				g.st.push(valueLongConst, c.Value)
			case *jclass.FloatConst:
				g.st.push(valueFloatConst, int64(math.Float32bits(c.Value)))
			case *jclass.DoubleConst:
				g.st.push(valueDoubleConst, int64(math.Float64bits(c.Value)))
			default:
				panic(fmt.Sprintf("%T const ldc", c))
			}

		case bytecode.Iload0, bytecode.Iload1, bytecode.Iload2, bytecode.Iload3:
			g.st.push(valueIntLocal, int64(op-bytecode.Iload0))
		case bytecode.Lload0, bytecode.Lload1, bytecode.Lload2, bytecode.Lload3:
			g.st.push(valueLongLocal, int64(op-bytecode.Lload0))
		case bytecode.Fload0, bytecode.Fload1, bytecode.Fload2, bytecode.Fload3:
			g.st.push(valueFloatLocal, int64(op-bytecode.Fload0))
		case bytecode.Dload0, bytecode.Dload1, bytecode.Dload2, bytecode.Dload3:
			g.st.push(valueDoubleLocal, int64(op-bytecode.Dload0))

		case bytecode.Istore0, bytecode.Istore1, bytecode.Istore2, bytecode.Istore3:
			dst := ir.Arg{Kind: ir.ArgReg, Value: int64(op - bytecode.Istore0)}
			g.out = append(g.out, ir.Inst{
				Dst:  dst,
				Kind: ir.InstLoad,
				Args: []ir.Arg{g.irArg(0)},
			})
			g.drop(1)

		case bytecode.Iadd:
			g.convertBinOp(ir.InstIadd)
		case bytecode.Ladd:
			g.convertBinOp(ir.InstLadd)
		case bytecode.Fadd:
			g.convertBinOp(ir.InstFadd)
		case bytecode.Dadd:
			g.convertBinOp(ir.InstDadd)
		case bytecode.Isub:
			g.convertBinOp(ir.InstIsub)

		case bytecode.Ineg:
			tmp := g.nextTmp()
			dst := ir.Arg{Kind: ir.ArgReg, Value: tmp + g.tmpOffset}
			g.out = append(g.out, ir.Inst{
				Dst:  dst,
				Kind: ir.InstIneg,
				Args: []ir.Arg{g.irArg(0)},
			})
			g.drop(1)
			g.st.push(valueTmp, tmp)

		case bytecode.Iinc:
			index := code[pc+1]
			delta := int8(code[pc+2])
			dst := ir.Arg{Kind: ir.ArgReg, Value: int64(index)}
			g.out = append(g.out, ir.Inst{
				Dst:  dst,
				Kind: ir.InstIadd,
				Args: []ir.Arg{
					dst,
					ir.Arg{Kind: ir.ArgIntConst, Value: int64(delta)},
				},
			})

		case bytecode.L2i:
			g.convertConv(ir.InstConvL2I)
		case bytecode.F2i:
			g.convertConv(ir.InstConvF2I)
		case bytecode.D2i:
			g.convertConv(ir.InstConvD2I)

		case bytecode.Lcmp:
			g.convertCmp(ir.InstLcmp)

		case bytecode.Ifeq:
			g.convertCondJump(code, pc, ir.InstJumpEqual)
		case bytecode.Ifne:
			g.convertCondJump(code, pc, ir.InstJumpNotEqual)
		case bytecode.Ificmpge:
			g.convertCmp(ir.InstIcmp)
			g.convertCondJump(code, pc, ir.InstJumpGtEq)

		case bytecode.Goto:
			ib1 := int16(code[pc+1])
			ib2 := int16(code[pc+2])
			inst := ir.Inst{
				Kind: ir.InstJump,
				Args: []ir.Arg{
					ir.Arg{Kind: ir.ArgBranch, Value: int64(ib1<<8 + ib2)},
				},
			}
			g.out = append(g.out, inst)
			g.toResolve = append(g.toResolve, unresolvedBranch{
				pc:     int32(pc),
				branch: &inst.Args[0].Value,
			})

		case bytecode.Ireturn:
			g.out = append(g.out, ir.Inst{
				Kind: ir.InstRet,
				Args: []ir.Arg{g.irArg(0)},
			})
			g.drop(1)
		case bytecode.Return:
			g.out = append(g.out, ir.Inst{
				Kind: ir.InstRet,
			})

		default:
			panic(fmt.Sprintf("unhandled op=%[1]d (0x%[1]x)", code[pc]))
		}

		pc += int(bytecode.OpWidth[op])
	}

	for _, u := range g.toResolve {
		pc := u.pc
		branch := int32(*u.branch)
		index, ok := pc2index[pc+branch]
		if !ok {
			panic(fmt.Sprintf("can't resolve branch with pc=%d and branch=%d", pc, branch))
		}
		*u.branch = int64(index)
	}

	g.out[0].Flags.SetJumpTarget(true)
	for _, inst := range g.out {
		switch inst.Kind {
		case ir.InstJumpEqual, ir.InstJumpNotEqual:
			index := inst.Args[0].Value
			g.out[index].Flags.SetJumpTarget(true)
		}
	}

	prevIsBranch := false
	for i, inst := range g.out {
		isLeader := i == 0 || prevIsBranch || inst.Flags.IsJumpTarget()
		if isLeader {
			g.out[i].Flags.SetBlockLead(true)
		}
		prevIsBranch = isJump(inst)
	}

	out := make([]ir.Inst, len(g.out))
	copy(out, g.out)
	return out
}

func (g *generator) convertCondJump(code []byte, pc int, kind ir.InstKind) {
	ib1 := int16(code[pc+1])
	ib2 := int16(code[pc+2])
	inst := ir.Inst{
		Kind: kind,
		Args: []ir.Arg{
			ir.Arg{Kind: ir.ArgBranch, Value: int64(ib1<<8 + ib2)},
			ir.Arg{Kind: ir.ArgFlags},
		},
	}
	g.out = append(g.out, inst)
	g.toResolve = append(g.toResolve, unresolvedBranch{
		pc:     int32(pc),
		branch: &inst.Args[0].Value,
	})

	if g.st.top().kind != valueFlags {
		panic(fmt.Sprintf("%s arg is not flags", kind))
	}
	g.drop(1)
}

func (g *generator) convertCmp(kind ir.InstKind) {
	g.out = append(g.out, ir.Inst{
		Dst:  ir.Arg{Kind: ir.ArgFlags},
		Kind: kind,
		Args: []ir.Arg{g.irArg(1), g.irArg(0)},
	})
	g.drop(2)
	g.st.push(valueFlags, 0)
}

func (g *generator) convertConv(kind ir.InstKind) {
	tmp := g.nextTmp()
	dst := ir.Arg{Kind: ir.ArgReg, Value: tmp + g.tmpOffset}
	g.out = append(g.out, ir.Inst{
		Dst:  dst,
		Kind: kind,
		Args: []ir.Arg{g.irArg(0)},
	})
	g.drop(1)
	g.st.push(valueTmp, tmp)
}

func (g *generator) convertBinOp(kind ir.InstKind) {
	tmp := g.nextTmp()
	dst := ir.Arg{Kind: ir.ArgReg, Value: tmp + g.tmpOffset}
	g.out = append(g.out, ir.Inst{
		Dst:  dst,
		Kind: kind,
		Args: []ir.Arg{g.irArg(1), g.irArg(0)},
	})
	g.drop(2)
	g.st.push(valueTmp, tmp)
}
